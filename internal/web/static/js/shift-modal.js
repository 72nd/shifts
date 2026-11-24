import { Modal } from './modal.js';
import {
    parseDate, parseDateTime, serializeDateTime, toInputValue, fromInputValue,
    addMinutes, addDays, logicalDay, daysBetween, minutesOfDay,
} from './datetime.js';

// Minutes after midnight for an "HH:MM" boundary string; 0 if invalid.
function boundaryToMinutes(boundary) {
    const m = /^(\d{1,2}):(\d{2})/.exec(boundary || '');
    if (!m) return 0;
    return Math.min(+m[1] * 60 + +m[2], 24 * 60 - 1);
}

export class ShiftModal extends Modal {
    constructor(planStartDate, shiftTypes, planToken, adminToken, onSuccess, planDayBoundary = '00:00') {
        super('shift_modal');
        this.planStartDate = planStartDate;
        this.shiftTypes = shiftTypes;
        this.planToken = planToken;
        this.adminToken = adminToken;
        this.onSuccess = onSuccess;
        this.planDayBoundary = boundaryToMinutes(planDayBoundary);
        this.reset();
    }

    reset() {
        super.clearError();
        this.errors = {}; // Field-level validation errors
        this.currentShift = null;
        this.currentShiftType = null;
        this.isDuplicating = false;
        this.type_id = '';
        this.start_time = '';
        this.duration = 60;
        this.required = 1;
        this.comment_de = '';
        this.comment_en = '';
        this.consecutive_shifts = 1;
        this.relative_day = 1;
        this.isUnlimited = false;
    }

    openWith(shift = null, type = null, isDuplicating = false) {
        this.reset();
        this.currentShift = isDuplicating ? null : shift;
        this.currentShiftType = type;
        this.isDuplicating = isDuplicating;

        if (shift) {
            // Edit or duplicate mode
            const startTime = parseDateTime(shift.start_time);
            if (!startTime) {
                console.error('Invalid shift start_time:', shift.start_time);
                this.setServerError('Invalid shift date format');
                return;
            }

            this.type_id = type.id;
            this.duration = shift.duration;
            this.required = shift.required;
            this.isUnlimited = shift.required === -1;
            this.comment_de = shift.comment_de || '';
            this.comment_en = shift.comment_en || '';

            // For duplication, move the start time on by the duration
            const start = isDuplicating ? addMinutes(startTime, shift.duration) : startTime;
            this.start_time = toInputValue(start);
            this.relative_day = this.getRelativeDayFromDate(start);
        } else {
            // Create mode: first plan day at 10:00
            const planStart = parseDate(this.planStartDate);
            if (!planStart) {
                console.error('Invalid plan start date:', this.planStartDate);
                this.setServerError('Invalid plan start date');
                return;
            }

            this.type_id = type ? type.id : '';
            this.start_time = toInputValue(addMinutes(planStart, 10 * 60));
        }

        this.open();
    }

    validate() {
        this.errors = {};
        let isValid = true;

        // Validate required fields
        if (!this.start_time) {
            this.errors.start_time = AlpineI18n.t('shift_plans.view.error_start_time_required');
            isValid = false;
        }

        const duration = parseInt(this.duration);
        if (isNaN(duration) || duration < 1) {
            this.errors.duration = AlpineI18n.t('shift_plans.view.error_duration_required');
            isValid = false;
        }

        // Only validate required participants if not unlimited
        if (!this.isUnlimited) {
            const required = parseInt(this.required);
            if (isNaN(required) || required < 1) {
                this.errors.required = AlpineI18n.t('shift_plans.view.error_required_participants_required');
                isValid = false;
            }
        }

        // Validate shift type selection when creating a new shift
        if (!this.currentShift && !this.type_id) {
            this.errors.type_id = AlpineI18n.t('shift_plans.view.error_shift_type_required');
            isValid = false;
        }

        return isValid;
    }

    async submit() {
        if (!this.validate()) return false;

        // Find the current shift type based on type_id
        const typeId = this.currentShift ? this.currentShiftType.id : this.type_id;
        const type = this.shiftTypes.find(t => t.id === parseInt(typeId));
        
        if (!type) {
            this.errors.type_id = AlpineI18n.t('shift_plans.view.error_invalid_shift_type');
            return false;
        }

        let success = false;
        if (this.currentShift) {
            success = await this.updateShift(type);
        } else {
            success = await this.createShifts(type);
        }

        if (success) {
            this.close();
            return true;
        }
        
        return false;
    }

    async updateShift(type) {
        // Validate the start time before sending
        const startTime = fromInputValue(this.start_time);
        if (!startTime) {
            console.error('Invalid start time for shift update:', this.start_time);
            this.setServerError('Invalid start time format');
            return false;
        }
        
        const url = `/api/plans/${this.planToken}/shifts/${this.currentShift.id}`;
        const options = {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
                'X-Admin-Token': this.adminToken
            },
            body: JSON.stringify({
                start_time: serializeDateTime(startTime),
                duration: parseInt(this.duration),
                required: this.isUnlimited ? -1 : parseInt(this.required),
                comment_de: this.comment_de,
                comment_en: this.comment_en,
                type_id: type.id
            })
        };
        
        const errorMessage = AlpineI18n.t('shift_plans.view.error_updating_shift');
        
        const { data } = await this.sendRequest(url, options, errorMessage);
        
        if (data !== null) {
            // Create updated shift object
            const updatedShift = {
                ...this.currentShift,
                start_time: serializeDateTime(startTime),
                duration: parseInt(this.duration),
                required: this.isUnlimited ? -1 : parseInt(this.required),
                comment_de: this.comment_de,
                comment_en: this.comment_en,
                type_id: type.id
            };

            // Notify about successful update
            if (this.onSuccess) {
                this.onSuccess([updatedShift]);
            }
            
            return true;
        }
        
        return false;
    }

    async createShifts(type) {
        const numShifts = parseInt(this.consecutive_shifts) || 1;
        const baseStartTime = fromInputValue(this.start_time);

        // Validate the base start time
        if (!baseStartTime) {
            console.error('Invalid start time for shift creation:', this.start_time);
            this.setServerError('Invalid start time format');
            return false;
        }
        
        const duration = parseInt(this.duration);
        const createdShifts = [];

        try {
            // Create all shifts in sequence
            for (let i = 0; i < numShifts; i++) {
                const shiftStartTime = addMinutes(baseStartTime, i * duration);

                const url = `/api/plans/${this.planToken}/shifts`;
                const options = {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'X-Admin-Token': this.adminToken
                    },
                    body: JSON.stringify({
                        type_id: type.id,
                        start_time: serializeDateTime(shiftStartTime),
                        duration: duration,
                        required: this.isUnlimited ? -1 : parseInt(this.required),
                        comment_de: this.comment_de,
                        comment_en: this.comment_en
                    })
                };
                
                const errorMessage = AlpineI18n.t('shift_plans.view.error_creating_shift');
                
                const { data } = await this.sendRequest(url, options, errorMessage);
                
                if (data === null) {
                    // If any shift creation fails, return false
                    return false;
                }
                
                createdShifts.push(data);
            }

            // Notify about successful creation
            if (this.onSuccess) {
                this.onSuccess(createdShifts);
            }
            
            return true;
        } catch (error) {
            this.setServerError(
                AlpineI18n.t('shift_plans.view.error_creating_shift'),
                { 
                    message: error.message,
                    stack: error.stack,
                    type: 'batch_creation_error',
                    createdShifts: createdShifts.length
                }
            );
            return false;
        }
    }

    getRelativeDayFromDate(date) {
        const planStart = parseDate(this.planStartDate);
        if (!planStart || !date) {
            console.warn('Invalid date detected in getRelativeDayFromDate:', {
                planStartDate: this.planStartDate,
                date: date
            });
            return 1; // Default to day 1
        }
        return daysBetween(planStart, logicalDay(date, this.planDayBoundary)) + 1;
    }

    updateStartTimeFromRelativeDay() {
        const planStart = parseDate(this.planStartDate);
        if (!planStart) {
            console.error('Invalid plan start date:', this.planStartDate);
            return; // Don't update if plan start date is invalid
        }

        const day = addDays(planStart, (parseInt(this.relative_day) || 1) - 1);
        const current = fromInputValue(this.start_time);
        // Keep the time of day; fall back to 09:00 if the field is empty or invalid
        const minutes = current ? minutesOfDay(current) : 9 * 60;
        this.start_time = toInputValue(addMinutes(day, minutes));
    }

    toggleUnlimited() {
        if (this.isUnlimited) {
            this.required = -1;
        } else {
            this.required = 1;
        }
    }
} 
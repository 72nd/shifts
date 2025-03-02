import { Modal } from './modal.js';

export class ShiftModal extends Modal {
    constructor(planStartDate, shiftTypes, planToken, adminToken, onSuccess) {
        super('shift_modal');
        this.planStartDate = planStartDate;
        this.shiftTypes = shiftTypes;
        this.planToken = planToken;
        this.adminToken = adminToken;
        this.onSuccess = onSuccess;
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
    }

    openWith(shift = null, type = null, isDuplicating = false) {
        this.reset();
        this.currentShift = isDuplicating ? null : shift;
        this.currentShiftType = type;
        this.isDuplicating = isDuplicating;

        if (shift) {
            // Edit or duplicate mode
            const startTime = new Date(shift.start_time);
            startTime.setMinutes(startTime.getMinutes() - startTime.getTimezoneOffset());
            
            this.type_id = type.id;
            this.start_time = startTime.toISOString().slice(0, 16);
            this.duration = shift.duration;
            this.required = shift.required;
            this.comment_de = shift.comment_de || '';
            this.comment_en = shift.comment_en || '';
            this.relative_day = this.getRelativeDayFromDate(startTime);

            if (isDuplicating) {
                // For duplication, increment the start time by the duration
                startTime.setMinutes(startTime.getMinutes() + shift.duration);
                this.start_time = startTime.toISOString().slice(0, 16);
                this.relative_day = this.getRelativeDayFromDate(startTime);
            }
        } else {
            // Create mode
            const startDate = new Date(this.planStartDate);
            startDate.setHours(10, 0, 0, 0);
            startDate.setMinutes(startDate.getMinutes() - startDate.getTimezoneOffset());
            
            this.type_id = type ? type.id : '';
            this.start_time = startDate.toISOString().slice(0, 16);
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

        const required = parseInt(this.required);
        if (isNaN(required) || required < 1) {
            this.errors.required = AlpineI18n.t('shift_plans.view.error_required_participants_required');
            isValid = false;
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
            this.setServerError(
                AlpineI18n.t('shift_plans.view.error_invalid_shift_type'),
                { message: 'Invalid shift type selected' }
            );
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
        const url = `/api/plans/${this.planToken}/shifts/${this.currentShift.id}`;
        const options = {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
                'X-Admin-Token': this.adminToken
            },
            body: JSON.stringify({
                start_time: new Date(this.start_time).toISOString(),
                duration: parseInt(this.duration),
                required: parseInt(this.required),
                comment_de: this.comment_de,
                comment_en: this.comment_en,
                type_id: type.id
            })
        };
        
        const errorMessage = AlpineI18n.t('shift_plans.view.error_updating_shift');
        
        const result = await this.sendRequest(url, options, errorMessage);
        
        if (result !== null) {
            // Create updated shift object
            const updatedShift = {
                ...this.currentShift,
                start_time: new Date(this.start_time).toISOString(),
                duration: parseInt(this.duration),
                required: parseInt(this.required),
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
        const baseStartTime = new Date(this.start_time);
        const duration = parseInt(this.duration);
        const createdShifts = [];

        try {
            // Create all shifts in sequence
            for (let i = 0; i < numShifts; i++) {
                const shiftStartTime = new Date(baseStartTime);
                shiftStartTime.setMinutes(shiftStartTime.getMinutes() + (i * duration));

                const url = `/api/plans/${this.planToken}/shifts`;
                const options = {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'X-Admin-Token': this.adminToken
                    },
                    body: JSON.stringify({
                        type_id: type.id,
                        start_time: shiftStartTime.toISOString(),
                        duration: duration,
                        required: parseInt(this.required),
                        comment_de: this.comment_de,
                        comment_en: this.comment_en
                    })
                };
                
                const errorMessage = AlpineI18n.t('shift_plans.view.error_creating_shift');
                
                const result = await this.sendRequest(url, options, errorMessage);
                
                if (result === null) {
                    // If any shift creation fails, return false
                    return false;
                }
                
                createdShifts.push(result);
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
        const planStart = new Date(this.planStartDate);
        const timeDiff = date.getTime() - planStart.getTime();
        return Math.floor(timeDiff / (1000 * 60 * 60 * 24)) + 1;
    }

    updateStartTimeFromRelativeDay() {
        const planStart = new Date(this.planStartDate);
        const currentStartTime = new Date(this.start_time);
        const hours = currentStartTime.getHours();
        const minutes = currentStartTime.getMinutes();

        planStart.setDate(planStart.getDate() + (this.relative_day - 1));
        planStart.setHours(hours, minutes, 0, 0);
        
        this.start_time = planStart.toISOString().slice(0, 16);
    }
} 
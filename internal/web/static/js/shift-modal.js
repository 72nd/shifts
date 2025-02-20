import { Modal } from './modal.js';

export class ShiftModal extends Modal {
    constructor(planStartDate, shiftTypes) {
        super('shift_modal');
        this.planStartDate = planStartDate;
        this.shiftTypes = shiftTypes;
        this.reset();
    }

    reset() {
        this.currentShift = null;
        this.currentShiftType = null;
        this.type_id = '';
        this.start_time = '';
        this.duration = 60;
        this.required = 1;
        this.comment_de = '';
        this.comment_en = '';
        this.consecutive_shifts = 1;
        this.relative_day = 1;
    }

    openWith(shift = null, type = null) {
        this.reset();
        this.currentShift = shift;
        this.currentShiftType = type;

        if (shift) {
            // Edit mode
            const startTime = new Date(shift.start_time);
            startTime.setMinutes(startTime.getMinutes() - startTime.getTimezoneOffset());
            
            this.type_id = type.id;
            this.start_time = startTime.toISOString().slice(0, 16);
            this.duration = shift.duration;
            this.required = shift.required;
            this.comment_de = shift.comment_de || '';
            this.comment_en = shift.comment_en || '';
            this.relative_day = this.getRelativeDayFromDate(startTime);
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
        if (!this.validate()) return;

        try {
            // Find the current shift type based on type_id
            const typeId = this.currentShift ? this.currentShiftType.id : this.type_id;
            const type = this.shiftTypes.find(t => t.id === parseInt(typeId));
            
            if (!type) {
                this.errors.general = AlpineI18n.t('shift_plans.view.error_invalid_shift_type');
                return;
            }

            if (this.currentShift) {
                await this.updateShift(type);
            } else {
                await this.createShifts(type);
            }

            // Signal successful completion
            this.close();
            return true;
        } catch (error) {
            console.error('Error saving shift:', error);
            this.errors.general = AlpineI18n.t(
                this.currentShift 
                    ? 'shift_plans.view.error_updating_shift'
                    : 'shift_plans.view.error_creating_shift'
            );
            return false;
        }
    }

    async updateShift(type) {
        const url = `/api/shift-plans/${this.planToken}/types/${type.id}/shifts/${this.currentShift.id}`;
        const response = await fetch(url, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
                'X-Admin-Token': this.adminToken
            },
            body: JSON.stringify({
                start_time: this.start_time,
                duration: parseInt(this.duration),
                required: parseInt(this.required),
                comment_de: this.comment_de,
                comment_en: this.comment_en
            })
        });

        if (!response.ok) throw new Error('Failed to update shift');
    }

    async createShifts(type) {
        const numShifts = parseInt(this.consecutive_shifts) || 1;
        const baseStartTime = new Date(this.start_time);
        const duration = parseInt(this.duration);

        // Create all shifts in sequence
        for (let i = 0; i < numShifts; i++) {
            const shiftStartTime = new Date(baseStartTime);
            shiftStartTime.setMinutes(shiftStartTime.getMinutes() + (i * duration));

            const response = await fetch(`/api/shift-plans/${this.planToken}/types/${type.id}/shifts`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Admin-Token': this.adminToken
                },
                body: JSON.stringify({
                    start_time: shiftStartTime.toISOString().slice(0, 16),
                    duration: duration,
                    required: parseInt(this.required),
                    comment_de: this.comment_de,
                    comment_en: this.comment_en
                })
            });

            if (!response.ok) throw new Error('Failed to create shift');
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

    setTokens(planToken, adminToken) {
        this.planToken = planToken;
        this.adminToken = adminToken;
    }
} 
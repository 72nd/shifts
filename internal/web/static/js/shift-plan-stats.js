document.addEventListener('alpine:init', () => {
    Alpine.data('shiftPlanStats', () => ({
        // State
        plan: null,
        stats: null,
        isLoading: true,
        error: null,
        adminToken: '',
        selectedHelper: null,
        
        // Initialize
        init() {
            // Get token from URL
            const pathParts = window.location.pathname.split('/');
            const token = pathParts[pathParts.length - 2]; // Token is now second-to-last part
            
            // Try to load admin token from localStorage
            this.adminToken = localStorage.getItem(`admin_token_${token}`);
            
            // Load data
            this.loadStats(token);
        },
        
        // Load statistics data
        async loadStats(token) {
            this.isLoading = true;
            this.error = null;
            
            try {
                const headers = {};
                if (this.adminToken) {
                    headers['X-Admin-Token'] = this.adminToken;
                }
                
                // Load plan and stats data
                const [planData, statsData] = await Promise.all([
                    fetch(`/api/plans/${token}`, { headers }).then(r => r.json()),
                    fetch(`/api/plans/${token}/stats`, { headers }).then(r => r.json())
                ]);
                
                this.plan = planData;
                this.stats = statsData;
                
            } catch (error) {
                console.error('Error loading stats:', error);
                this.error = error.message;
            } finally {
                this.isLoading = false;
            }
        },
        
        // Computed properties for statistics
        getTotalStats() {
            if (!this.shiftTypes) return null;
            
            const now = new Date();
            const stats = {
                totalShifts: 0,
                totalRequired: 0,
                totalFilled: 0,
                totalHours: 0,
                filledHours: 0,
                pastHours: 0,
                futureHours: 0,
                uniqueHelpers: new Set(),
                openShifts: 0,
                openPositions: 0
            };
            
            this.shiftTypes.forEach(type => {
                if (!type.shifts) return;
                
                type.shifts.forEach(shift => {
                    const startTime = new Date(shift.start_time);
                    const endTime = new Date(startTime.getTime() + shift.duration * 60000);
                    const shiftHours = shift.duration / 60;
                    const filledPositions = shift.participants?.length || 0;
                    
                    stats.totalShifts++;
                    stats.totalRequired += shift.required;
                    stats.totalFilled += filledPositions;
                    
                    stats.totalHours += shiftHours * shift.required;
                    stats.filledHours += shiftHours * filledPositions;
                    
                    // Calculate past and future hours
                    if (endTime < now) {
                        stats.pastHours += shiftHours * filledPositions;
                    } else if (startTime > now) {
                        stats.futureHours += shiftHours * (shift.required - filledPositions);
                    } else {
                        // Shift is currently ongoing
                        const progressPercent = (now - startTime) / (endTime - startTime);
                        stats.pastHours += shiftHours * filledPositions * progressPercent;
                        stats.futureHours += shiftHours * (shift.required - filledPositions) * (1 - progressPercent);
                    }
                    
                    if (filledPositions < shift.required) {
                        stats.openShifts++;
                        stats.openPositions += shift.required - filledPositions;
                    }
                    
                    shift.participants?.forEach(p => stats.uniqueHelpers.add(p.id));
                });
            });
            
            return {
                ...stats,
                uniqueHelpers: stats.uniqueHelpers.size,
                percentageFilled: Math.round((stats.totalFilled / stats.totalRequired) * 100)
            };
        },
        
        // Get helper rankings
        getHelperRankings() {
            if (!this.shiftTypes) return [];
            
            const helperStats = new Map();
            const now = new Date();
            
            this.shiftTypes.forEach(type => {
                type.shifts?.forEach(shift => {
                    const startTime = new Date(shift.start_time);
                    const endTime = new Date(startTime.getTime() + shift.duration * 60000);
                    const shiftHours = shift.duration / 60;
                    
                    shift.participants?.forEach(participant => {
                        const stats = helperStats.get(participant.id) || {
                            id: participant.id,
                            name: participant.name,
                            shifts: 0,
                            hours: 0,
                            pastHours: 0,
                            futureHours: 0,
                            shiftDetails: []
                        };
                        
                        stats.shifts++;
                        
                        if (endTime < now) {
                            stats.pastHours += shiftHours;
                        } else if (startTime > now) {
                            stats.futureHours += shiftHours;
                        } else {
                            // Shift is currently ongoing
                            const progressPercent = (now - startTime) / (endTime - startTime);
                            stats.pastHours += shiftHours * progressPercent;
                            stats.futureHours += shiftHours * (1 - progressPercent);
                        }
                        
                        stats.hours = stats.pastHours + stats.futureHours;
                        
                        // Add shift details for the helper details view
                        stats.shiftDetails.push({
                            date: startTime,
                            duration: shiftHours,
                            type: type.title_de || type.title_en,
                            isPast: endTime < now,
                            isCurrent: startTime <= now && endTime >= now
                        });
                        
                        helperStats.set(participant.id, stats);
                    });
                });
            });
            
            return Array.from(helperStats.values())
                .sort((a, b) => b.hours - a.hours);
        },
        
        // Get statistics by type
        getTypeStats() {
            if (!this.shiftTypes) return [];
            
            return this.shiftTypes.map(type => {
                const stats = {
                    id: type.id,
                    title: this.$locale() === 'de' ? type.title_de : type.title_en,
                    totalShifts: 0,
                    totalRequired: 0,
                    totalFilled: 0,
                    totalHours: 0,
                    filledHours: 0,
                    openPositions: 0
                };
                
                type.shifts?.forEach(shift => {
                    stats.totalShifts++;
                    stats.totalRequired += shift.required;
                    stats.totalFilled += shift.participants?.length || 0;
                    
                    const shiftHours = shift.duration / 60;
                    stats.totalHours += shiftHours * shift.required;
                    stats.filledHours += shiftHours * (shift.participants?.length || 0);
                    
                    if ((shift.participants?.length || 0) < shift.required) {
                        stats.openPositions += shift.required - (shift.participants?.length || 0);
                    }
                });
                
                stats.percentageFilled = Math.round((stats.totalFilled / stats.totalRequired) * 100);
                
                return stats;
            });
        },
        
        // Get statistics by day
        getDayStats() {
            if (!this.shiftTypes) return [];
            
            const dayStats = new Map();
            
            this.shiftTypes.forEach(type => {
                type.shifts?.forEach(shift => {
                    const date = new Date(shift.start_time).toISOString().split('T')[0];
                    
                    const stats = dayStats.get(date) || {
                        date,
                        totalShifts: 0,
                        totalRequired: 0,
                        totalFilled: 0,
                        totalHours: 0,
                        filledHours: 0,
                        openPositions: 0
                    };
                    
                    stats.totalShifts++;
                    stats.totalRequired += shift.required;
                    stats.totalFilled += shift.participants?.length || 0;
                    
                    const shiftHours = shift.duration / 60;
                    stats.totalHours += shiftHours * shift.required;
                    stats.filledHours += shiftHours * (shift.participants?.length || 0);
                    
                    if ((shift.participants?.length || 0) < shift.required) {
                        stats.openPositions += shift.required - (shift.participants?.length || 0);
                    }
                    
                    dayStats.set(date, stats);
                });
            });
            
            return Array.from(dayStats.values())
                .sort((a, b) => a.date.localeCompare(b.date))
                .map(stats => ({
                    ...stats,
                    percentageFilled: Math.round((stats.totalFilled / stats.totalRequired) * 100)
                }));
        },
        
        // Formatting helpers
        formatDate(date) {
            return new Date(date).toLocaleDateString(this.$locale() === 'de' ? 'de-DE' : 'en-US', {
                weekday: 'short',
                year: 'numeric',
                month: 'short',
                day: 'numeric'
            });
        },
        
        formatHours(hours) {
            if (hours === undefined || hours === null) return '0';
            return hours.toFixed(1);
        },

        // Get selected helper details
        getSelectedHelperDetails() {
            if (!this.selectedHelper || !this.stats) return null;
            return this.stats.helper_stats.find(h => h.name === this.selectedHelper);
        }
    }));
}); 
import {
    parseDate, parseDateTime, addMinutes, logicalDay, serializeDate,
    formatDate as formatFloatingDate, nowInZone
} from './datetime.js';

document.addEventListener('alpine:init', () => {
    Alpine.data('shiftPlanStats', () => ({
        // State
        plan: null,
        stats: null,
        isLoading: true,
        error: null,
        adminToken: '',
        selectedHelper: null,
        isHelperRankingsExpanded: false,
        helperRankingsCollapsedLimit: 5,
        selectedParticipantFilter: '',
        expandedDays: {}, // Track which days are expanded: { dayKey: true }
        daySortConfig: {}, // Map<dayKey, { column: string, direction: 'asc' | 'desc' }>
        
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
        
        // Initialize expanded days after stats are loaded
        initExpandedDays() {
            if (!this.stats?.helper_stats) return;
            
            // Expand all days by default if not already set
            const dailyStats = this.getDailyParticipantStats();
            const newExpandedDays = { ...this.expandedDays };
            dailyStats.forEach(dayStat => {
                if (!(dayStat.dayKey in newExpandedDays)) {
                    newExpandedDays[dayStat.dayKey] = true;
                }
            });
            this.expandedDays = newExpandedDays;
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
                
                // Initialize expanded days after stats are loaded
                setTimeout(() => {
                    this.initExpandedDays();
                }, 0);
                
            } catch (error) {
                console.error('Error loading stats:', error);
                this.error = error.message;
            } finally {
                this.isLoading = false;
            }
        },
        
        // Current wall clock in the plan's time zone (floating Date)
        nowInPlanZone() {
            return nowInZone(this.plan?.time_zone);
        },

        // Daily boundary of the plan in minutes after midnight ("HH:MM")
        getBoundaryMinutes() {
            const m = /^(\d{1,2}):(\d{2})/.exec(this.plan?.day_boundary_time || '');
            return m ? Number(m[1]) * 60 + Number(m[2]) : 0;
        },

        // Start and end of a shift as floating Dates, or null if unparsable
        getShiftSpan(shift) {
            const startTime = parseDateTime(shift.start_time);
            if (!startTime) return null;
            return { startTime, endTime: addMinutes(startTime, shift.duration) };
        },

        // Computed properties for statistics
        getTotalStats() {
            if (!this.shiftTypes) return null;
            
            const now = this.nowInPlanZone();
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
                    const span = this.getShiftSpan(shift);
                    if (!span) return;
                    const { startTime, endTime } = span;
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
            const now = this.nowInPlanZone();
            
            this.shiftTypes.forEach(type => {
                type.shifts?.forEach(shift => {
                    const span = this.getShiftSpan(shift);
                    if (!span) return;
                    const { startTime, endTime } = span;
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
                    const start = parseDateTime(shift.start_time);
                    if (!start) return;
                    const date = serializeDate(logicalDay(start, this.getBoundaryMinutes()));
                    
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
            return formatFloatingDate(date, this.$locale(), {
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
        },
        
        // Get visible helper rankings (for collapsed view)
        getVisibleHelperRankings() {
            if (!this.stats?.helper_stats) return [];
            return this.stats.helper_stats.slice(0, this.helperRankingsCollapsedLimit);
        },
        
        // Get daily participant statistics grouped by day
        getDailyParticipantStats() {
            if (!this.stats?.helper_stats) return [];
            
            // Map to store daily stats: dayKey -> { date, participants: Map<name, {shifts, types, hours}> }
            const dailyMap = new Map();
            
            // Process each helper's shift details
            this.stats.helper_stats.forEach(helper => {
                helper.shift_details?.forEach(shiftDetail => {
                    // Server sends the logical day as YYYY-MM-DD
                    const shiftDate = parseDate(shiftDetail.date);
                    if (!shiftDate) return;
                    const dayKey = serializeDate(shiftDate);
                    
                    // Get or create day entry
                    let dayStats = dailyMap.get(dayKey);
                    if (!dayStats) {
                        dayStats = {
                            date: shiftDate,
                            dayKey: dayKey,
                            participants: new Map()
                        };
                        dailyMap.set(dayKey, dayStats);
                    }
                    
                    // Get or create participant entry for this day
                    let participantStats = dayStats.participants.get(helper.name);
                    if (!participantStats) {
                        participantStats = {
                            name: helper.name,
                            shiftCount: 0,
                            shiftTypes: new Map(), // Map<type_name, count>
                            totalHours: 0
                        };
                        dayStats.participants.set(helper.name, participantStats);
                    }
                    
                    // Update participant stats
                    participantStats.shiftCount++;
                    const currentCount = participantStats.shiftTypes.get(shiftDetail.type_name) || 0;
                    participantStats.shiftTypes.set(shiftDetail.type_name, currentCount + 1);
                    participantStats.totalHours += shiftDetail.duration;
                });
            });
            
            // Convert to array and sort by date
            return Array.from(dailyMap.values())
                .sort((a, b) => a.date - b.date)
                .map(dayStats => ({
                    ...dayStats,
                    participants: Array.from(dayStats.participants.values())
                        .map(p => ({
                            ...p,
                            shiftTypes: Array.from(p.shiftTypes.entries())
                                .map(([typeName, count]) => ({ typeName, count }))
                                .sort((a, b) => a.typeName.localeCompare(b.typeName))
                        }))
                        .sort((a, b) => a.name.localeCompare(b.name))
                }));
        },
        
        // Get filtered daily participant stats
        getFilteredDailyParticipantStats() {
            const dailyStats = this.getDailyParticipantStats();
            const filter = this.selectedParticipantFilter.trim().toLowerCase();
            
            let filtered = dailyStats;
            
            // Apply participant filter
            if (filter) {
                filtered = dailyStats.map(dayStat => ({
                    ...dayStat,
                    participants: dayStat.participants.filter(p => 
                        p.name.toLowerCase().includes(filter)
                    )
                })).filter(dayStat => dayStat.participants.length > 0);
            }
            
            // Apply sorting for each day
            return filtered.map(dayStat => {
                const sortConfig = this.daySortConfig[dayStat.dayKey];
                if (!sortConfig) {
                    return dayStat;
                }
                
                const sortedParticipants = [...dayStat.participants].sort((a, b) => {
                    let aVal, bVal;
                    
                    switch (sortConfig.column) {
                        case 'name':
                            aVal = a.name.toLowerCase();
                            bVal = b.name.toLowerCase();
                            break;
                        case 'shifts':
                            aVal = a.shiftCount;
                            bVal = b.shiftCount;
                            break;
                        case 'hours':
                            aVal = a.totalHours;
                            bVal = b.totalHours;
                            break;
                        case 'types':
                            aVal = a.shiftTypes.length;
                            bVal = b.shiftTypes.length;
                            break;
                        default:
                            return 0;
                    }
                    
                    if (aVal < bVal) return sortConfig.direction === 'asc' ? -1 : 1;
                    if (aVal > bVal) return sortConfig.direction === 'asc' ? 1 : -1;
                    return 0;
                });
                
                return {
                    ...dayStat,
                    participants: sortedParticipants
                };
            });
        },
        
        // Toggle day expansion
        toggleDay(dayKey) {
            this.expandedDays[dayKey] = !this.expandedDays[dayKey];
            // Force reactivity
            this.expandedDays = { ...this.expandedDays };
        },
        
        // Check if day is expanded (defaults to true if not set)
        isDayExpanded(dayKey) {
            return this.expandedDays[dayKey] !== false;
        },
        
        // Handle column sorting
        sortColumn(dayKey, column) {
            const currentSort = this.daySortConfig[dayKey];
            
            if (currentSort && currentSort.column === column) {
                // Toggle direction if same column
                this.daySortConfig[dayKey] = {
                    column: column,
                    direction: currentSort.direction === 'asc' ? 'desc' : 'asc'
                };
            } else {
                // New column, default to ascending
                this.daySortConfig[dayKey] = {
                    column: column,
                    direction: 'asc'
                };
            }
            
            // Force reactivity
            this.daySortConfig = { ...this.daySortConfig };
        },
        
        // Get sort icon for column
        getSortIcon(dayKey, column) {
            const sortConfig = this.daySortConfig[dayKey];
            if (!sortConfig || sortConfig.column !== column) {
                return 'M7 16V4m0 0L3 8m4-4l4 4m6 0v12m0 0l4-4m-4 4l-4-4'; // Sort icon (both directions)
            }
            if (sortConfig.direction === 'asc') {
                return 'M5 15l7-7 7 7'; // Up arrow
            }
            return 'M19 9l-7 7-7-7'; // Down arrow
        },
        
        // Get helper rankings preview message
        getHelperRankingsPreviewMessage() {
            const total = this.stats?.helper_stats?.length || 0;
            if (total <= this.helperRankingsCollapsedLimit) {
                return '';
            }
            return `Showing top ${this.helperRankingsCollapsedLimit} (${total} total)`;
        }
    }));
}); 
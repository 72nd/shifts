/**
 * Lab tab: UI state of the coordination lab (prj/coordination-improvements-plan.md, 5.5 and 6.3).
 *
 * A draft is a list of target statuses (`ops`, see lab.js). The lab model is always
 * applyOpsLocally(liveModel, ops), so sign-ups of other people show up by themselves. Nothing is
 * stored until the person saves a named variant (server) or applies the draft (page apply()).
 *
 * The component is nested in x-data="coordinationPage" (wrapper in coordinate.html). Methods reach the page
 * through `this.$data` (`pg`); `labDay` and the URL state live in the page. No rule or fairness logic here:
 * everything shown comes from lab.js and the model.
 */
import { personKey } from '../stats/compute.js';
import { formatDate, formatTime, instantInZone } from '../datetime.js';
import { applyOpsLocally } from './compute.js';
import {
    normalizeOps, moveOps, removeOps, buildLabGrid, personDeltas, labSummary,
} from './lab.js';

const UNDO_LIMIT = 200;
const NAME_MAX = 80;        // variant name (validation.ValidateCoordinationDraft)
const PERSON_NAME_MAX = 20; // participant names (validation.MaxNameLength)
const LAB_SORTS = ['load', 'hours', 'experience', 'name'];

/** Search form of a name: no diacritics, lower case, single spaces. */
function searchKey(text) {
    return String(text || '').normalize('NFD').replace(/\p{Diacritic}/gu, '').normalize('NFC').trim().replace(/\s+/g, ' ').toLocaleLowerCase();
}

/** Order-independent form of normalized ops, to compare two drafts. */
function opsSignature(ops) {
    return JSON.stringify([...ops]
        .map(o => [o.shift_id, personKey(o.name), o.status])
        .sort((a, b) => a[0] - b[0] || a[1].localeCompare(b[1])));
}

document.addEventListener('alpine:init', () => {
    Alpine.data('coordLab', () => {
        // Memo of the derived values: recomputed only when an input object changes (identity)
        const memo = {};
        const cached = (name, deps, fn) => {
            const m = memo[name];
            if (m && m.deps.length === deps.length && deps.every((d, i) => d === m.deps[i])) return m.value;
            const value = fn();
            memo[name] = { deps, value };
            return value;
        };
        let afterConfirm = null; // what the confirm dialog does on "yes"

        return {
            ops: [],               // normalized ops of the draft
            undoStack: [],         // previous values of ops
            draftId: null,         // loaded variant (null = unsaved draft)
            draftName: '',
            savedOps: [],          // ops of the loaded variant (for "unsaved changes")
            drafts: [],            // variant summaries from the server
            draftsLoaded: false,
            draftsLoading: false,
            labBusy: false,        // saving or deleting a variant
            picked: null,          // keyboard "pick up": { key, name, from }
            dragging: null,        // mouse drag: { key, name, from }
            dragOver: null,        // shift id under the pointer while dragging
            announce: '',          // live region text
            selectedKey: null,
            extraNames: [],        // people added in the lab that have no assignment yet
            labQuery: '',
            labSort: 'load',
            newName: '',
            newNameError: '',
            // name dialog: 'save' | 'saveAs' | 'rename'
            nameMode: 'save',
            nameInput: '',
            nameError: '',
            confirmKind: '',       // 'discard' | 'delete'

            init() {
                this.$watch('tab', (t) => { if (t === 'lab') this.loadDrafts(); });
                if (this.pg.tab === 'lab') this.loadDrafts();
                window.addEventListener('beforeunload', (e) => {
                    if (!this.dirty) return;
                    e.preventDefault();
                    e.returnValue = '';
                });
            },

            // --- Page access and derived values ---

            get pg() {
                return this.$data;
            },

            get live() {
                return this.pg.model;
            },

            get normOps() {
                return cached('norm', [this.ops, this.live], () => normalizeOps(Alpine.raw(this.live), this.ops));
            },

            get labModel() {
                return cached('lab', [this.live, this.normOps], () => applyOpsLocally(Alpine.raw(this.live), this.normOps));
            },

            /** Unsaved changes compared with the loaded variant (or with the empty draft). */
            get dirty() {
                return opsSignature(this.normOps) !== opsSignature(normalizeOps(Alpine.raw(this.live), this.savedOps));
            },

            get opCount() {
                return this.normOps.length;
            },

            /** "shiftId|key" pairs touched by the ops (chip marker). */
            get changedPairs() {
                return cached('pairs', [this.normOps], () => new Set(this.normOps.map(o => `${o.shift_id}|${personKey(o.name)}`)));
            },

            /** Day shown by the grid: the chosen one if it has shifts, else the day of the selected shift, else the first day. */
            get currentLabDay() {
                const days = this.pg.dayOptions;
                if (this.pg.labDay && days.includes(this.pg.labDay)) return this.pg.labDay;
                return this.pg.selectedShift?.day || days[0] || '';
            },

            get hasPrevDay() {
                return this.pg.dayOptions.indexOf(this.currentLabDay) > 0;
            },

            get hasNextDay() {
                const days = this.pg.dayOptions;
                const i = days.indexOf(this.currentLabDay);
                return i >= 0 && i < days.length - 1;
            },

            setLabDay(day) {
                this.pg.labDay = day;
                this.pg.writeUrl();
            },

            shiftLabDay(n) {
                const days = this.pg.dayOptions;
                const i = days.indexOf(this.currentLabDay);
                const next = days[Math.min(days.length - 1, Math.max(0, i + n))];
                if (next) this.setLabDay(next);
            },

            get grid() {
                return cached('grid', [this.live, this.labModel, this.currentLabDay, this.changedPairs],
                    () => buildLabGrid(Alpine.raw(this.live), this.labModel, this.currentLabDay, this.changedPairs));
            },

            /** Minimum width of the grid: name column plus 7 rem per hour. */
            gridMinWidth() {
                return `calc(8rem + ${this.grid.hours.length * 7}rem)`;
            },

            /** Per person: live vs lab numbers (personDeltas). */
            get deltas() {
                return cached('deltas', [this.live, this.labModel], () => personDeltas(Alpine.raw(this.live), this.labModel));
            },

            /** Rows of the side panel: people of the lab model plus the ones added by name only. */
            get sideRows() {
                const deltas = this.deltas;
                const rows = [...deltas.values()];
                for (const name of this.extraNames) {
                    const key = personKey(name);
                    if (deltas.has(key)) continue;
                    const zero = { hours: 0, load: 0, popularScore: 0, warnings: [], perDayMin: new Map() };
                    rows.push({
                        key, name, level: -1, isNew: true, live: zero, lab: zero,
                        hoursDelta: 0, loadDelta: 0, scoreDelta: 0, perDay: new Map(), newWarnings: [],
                    });
                }
                const q = searchKey(this.labQuery);
                const list = q ? rows.filter(r => searchKey(r.name).includes(q)) : rows;
                const byName = (a, b) => a.name.localeCompare(b.name);
                const sorts = {
                    load: (a, b) => b.lab.load - a.lab.load || byName(a, b),
                    hours: (a, b) => b.lab.hours - a.lab.hours || byName(a, b),
                    experience: (a, b) => b.level - a.level || byName(a, b),
                    name: byName,
                };
                return list.sort(sorts[LAB_SORTS.includes(this.labSort) ? this.labSort : 'load']);
            },

            get searchEmpty() {
                return !!searchKey(this.labQuery) && this.sideRows.length === 0;
            },

            /** True if someone only exists as a name (no assignment yet): the hint "lost on reload". */
            get hasUnassignedNew() {
                const keys = this.deltas;
                return this.extraNames.some(n => !keys.has(personKey(n)));
            },

            /** The chosen person (row of sideRows with its numbers), or null. */
            get detail() {
                if (!this.selectedKey) return null;
                const all = this.deltas.get(this.selectedKey);
                if (all) return all;
                const name = this.extraNames.find(n => personKey(n) === this.selectedKey);
                return name ? this.sideRows.find(r => r.key === this.selectedKey) || null : null;
            },

            /** Rows of the detail table: minutes per day live vs lab, ascending by day. */
            detailDays(detail) {
                return [...detail.perDay.entries()]
                    .map(([day, v]) => ({ day, live: v.live, lab: v.lab }))
                    .sort((a, b) => a.day.localeCompare(b.day));
            },

            /** Wishes of the person on the shown day (lab model). */
            detailWishes(key) {
                return [...this.labModel.shifts.values()].filter(cs => cs.day === this.currentLabDay && cs.wishes.includes(key));
            },

            /** Texts of the lab warnings of a person (tooltip and screen readers). */
            warnText(list) {
                return list.map(w => this.pg.warningText(w)).join(', ');
            },

            /** Difference with sign ("+4", "−1.5"); '' if (nearly) zero. */
            delta(n) {
                if (Math.abs(n) < 0.05) return '';
                return (n > 0 ? '+' : '−') + this.pg.number(Math.abs(n));
            },

            /** Shift of the lab model as "Bar, Sa 4.7. 20:00–24:00". */
            labShiftLabel(id) {
                return this.pg.shiftLabel(this.labModel.shifts.get(id));
            },

            /** Header of a box: "3/4", with "+1" when over the need. */
            boxCount(b) {
                const need = b.required < 0 ? '∞' : b.required;
                const diff = b.required < 0 ? 0 : b.einsaetze - b.required;
                return `${b.einsaetze}/${need}${diff > 0 ? ` +${diff}` : diff < 0 ? ` −${-diff}` : ''}`;
            },

            /** Text of one op for the change list: "Anna Berg – Bar, Sa 10:00: eingeteilt". */
            opText(op) {
                const cs = this.live.shifts.get(op.shift_id);
                const where = cs ? `${this.pg.typeTitle(cs.type)}, ${this.pg.dayLabel(cs.start)} ${this.pg.time(cs.start)}` : String(op.shift_id);
                const status = this.$t(`coord.lab_status_${op.status === 'signed_up' ? 'assigned' : op.status}`);
                return `${op.name} – ${where}: ${status}`;
            },

            // --- Variants (server) ---

            /** "Name (12.07. 10:00, übernommen)" with the time in the plan's zone. */
            draftLabel(d) {
                const at = instantInZone(d.updated_at, this.pg.plan.time_zone);
                const lang = this.pg.lang();
                const when = at ? `${formatDate(at, lang, { day: 'numeric', month: 'numeric' })} ${formatTime(at, lang)}` : '';
                const extra = [when, d.applied_at ? this.$t('coord.lab_applied_mark') : ''].filter(Boolean).join(', ');
                return extra ? `${d.name} (${extra})` : d.name;
            },

            failLab(e, key, modal) {
                const err = new Error(e.status === 409 ? this.$t('coord.lab_error_limit') : this.$t(key));
                err.status = e.status;
                this.pg.fail(err, modal);
                if (modal) this.pg.modalError.errorDetails = { message: e.message, status: e.status };
            },

            async loadDrafts() {
                if (this.draftsLoaded || this.draftsLoading) return;
                this.draftsLoading = true;
                try {
                    this.drafts = (await this.pg.api('GET', '/coordination/drafts')) || [];
                    this.draftsLoaded = true;
                } catch (e) {
                    this.failLab(e, 'coord.lab_error_load', false);
                } finally {
                    this.draftsLoading = false;
                }
            },

            async refreshDrafts() {
                this.draftsLoaded = false;
                await this.loadDrafts();
            },

            /** Variant select changed: load it (after the question if there are unsaved changes). */
            onVariantChange(event) {
                const value = event.target.value;
                const current = this.draftId === null ? '' : String(this.draftId);
                if (value === current) return;
                const go = () => (value === '' ? this.startEmpty() : this.loadDraft(Number(value)));
                if (this.dirty) {
                    event.target.value = current; // stays until the question is answered
                    this.askConfirm('discard', go);
                } else {
                    go();
                }
            },

            startEmpty() {
                this.ops = [];
                this.undoStack = [];
                this.savedOps = [];
                this.draftId = null;
                this.draftName = '';
            },

            async loadDraft(id) {
                try {
                    const d = await this.pg.api('GET', `/coordination/drafts/${id}`);
                    this.savedOps = d.ops || [];
                    this.ops = normalizeOps(Alpine.raw(this.live), this.savedOps);
                    this.undoStack = [];
                    this.draftId = d.id;
                    this.draftName = d.name;
                } catch (e) {
                    this.failLab(e, 'coord.lab_error_load', false);
                    await this.refreshDrafts();
                }
            },

            /** "Speichern": replace the loaded variant, or ask for a name first. */
            async save() {
                if (this.draftId === null) return this.openName('save');
                await this.putDraft(this.draftName, this.normOps, false, false);
            },

            async putDraft(name, ops, applied, modal) {
                this.labBusy = true;
                try {
                    const d = await this.pg.api('PUT', `/coordination/drafts/${this.draftId}`, { name, ops, applied });
                    this.draftName = d.name;
                    this.savedOps = d.ops || ops;
                    await this.refreshDrafts();
                    return true;
                } catch (e) {
                    this.failLab(e, 'coord.lab_error_save', modal);
                    return false;
                } finally {
                    this.labBusy = false;
                }
            },

            openName(mode) {
                this.nameMode = mode;
                this.nameInput = mode === 'save' ? '' : this.draftName;
                this.nameError = '';
                this.pg.clearErrors();
                this.$refs.nameDlg.showModal();
                this.$nextTick(() => this.$refs.nameField?.focus());
            },

            closeName() {
                this.pg.clearErrors();
                this.$refs.nameDlg.close();
            },

            /** Name dialog: create (save, saveAs) or rename; the content of the variant stays what was saved. */
            async submitName() {
                const name = this.nameInput.trim().replace(/\s+/g, ' ');
                if (!name) {
                    this.nameError = this.$t('shift_plans.view.error_name_required');
                    return;
                }
                if ([...name].length > NAME_MAX) {
                    this.nameError = this.$t('shift_plans.view.error_name_too_long', { max: NAME_MAX });
                    return;
                }
                this.nameError = '';
                this.pg.clearErrors();
                if (this.nameMode === 'rename') {
                    if (await this.putDraft(name, normalizeOps(Alpine.raw(this.live), this.savedOps), false, true)) this.closeName();
                    return;
                }
                this.labBusy = true;
                try {
                    const d = await this.pg.api('POST', '/coordination/drafts', { name, ops: this.normOps });
                    this.draftId = d.id;
                    this.draftName = d.name;
                    this.savedOps = d.ops || this.normOps;
                    this.pg.showToast(this.$t('coord.lab_saved'), false);
                    await this.refreshDrafts();
                    this.closeName();
                } catch (e) {
                    this.failLab(e, 'coord.lab_error_save', true);
                } finally {
                    this.labBusy = false;
                }
            },

            askConfirm(kind, then) {
                this.confirmKind = kind;
                afterConfirm = then || null;
                this.pg.clearErrors();
                this.$refs.confirmDlg.showModal();
            },

            closeConfirm() {
                this.pg.clearErrors();
                afterConfirm = null;
                this.$refs.confirmDlg.close();
            },

            /** "Yes" of the confirm dialog. Deleting a variant keeps the draft in the tab (as an unsaved one). */
            async confirmYes() {
                if (this.confirmKind === 'delete') {
                    this.labBusy = true;
                    try {
                        await this.pg.api('DELETE', `/coordination/drafts/${this.draftId}`);
                        this.draftId = null;
                        this.draftName = '';
                        this.savedOps = [];
                        this.pg.showToast(this.$t('coord.lab_deleted'), false);
                        await this.refreshDrafts();
                        this.closeConfirm();
                    } catch (e) {
                        this.failLab(e, 'coord.lab_error_save', true);
                    } finally {
                        this.labBusy = false;
                    }
                    return;
                }
                const then = afterConfirm;
                this.closeConfirm();
                if (then) then();
            },

            /** "Verwerfen": back to the loaded variant (or to empty); one undo step. */
            discard() {
                if (!this.dirty) return;
                this.push();
                this.ops = normalizeOps(Alpine.raw(this.live), this.savedOps);
            },

            // --- Apply ---

            get summary() {
                return cached('summary', [this.live, this.labModel, this.normOps],
                    () => labSummary(Alpine.raw(this.live), this.labModel, this.normOps));
            },

            /** Names of people the lab puts into two simultaneous shifts (not those that already had it live). */
            get newOverlaps() {
                return [...this.deltas.values()]
                    .filter(d => d.newWarnings.some(w => w.kind === 'overlap'))
                    .map(d => d.name)
                    .sort((a, b) => a.localeCompare(b));
            },

            openApply() {
                if (!this.opCount) return;
                this.pg.clearErrors();
                this.$refs.applyDlg.showModal();
            },

            closeApply() {
                this.pg.clearErrors();
                this.$refs.applyDlg.close();
            },

            /** Applies the ops through the page (undo in the toast), marks the loaded variant as applied, empties the draft. */
            async confirmApply() {
                const ops = this.normOps;
                if (!ops.length) return;
                if (!(await this.pg.apply(ops, { modal: true, pending: 'lab' }))) return;
                if (this.draftId !== null) {
                    // The variant now holds exactly what was applied
                    await this.putDraft(this.draftName, ops, true, false);
                }
                this.startEmpty();
                this.extraNames = [];
                this.closeApply();
            },

            // --- Editing the draft ---

            push() {
                this.undoStack.push(this.ops);
                if (this.undoStack.length > UNDO_LIMIT) this.undoStack.shift();
            },

            /** Adds ops to the draft as one undo step. */
            commit(newOps) {
                if (!newOps.length) return;
                const next = normalizeOps(Alpine.raw(this.live), [...this.ops, ...newOps]);
                if (opsSignature(next) === opsSignature(this.ops)) return;
                this.push();
                this.ops = next;
            },

            undoLab() {
                if (!this.undoStack.length) return;
                this.ops = this.undoStack.pop();
            },

            /** Ctrl/Cmd+Z in the lab area (not while typing in a field), Esc cancels "picked up". */
            onKey(event) {
                if (event.key === 'Escape' && this.picked) {
                    this.picked = null;
                    this.announce = '';
                    return;
                }
                if ((event.ctrlKey || event.metaKey) && !event.shiftKey && event.key.toLowerCase() === 'z') {
                    if (['INPUT', 'TEXTAREA', 'SELECT'].includes(event.target.tagName)) return;
                    event.preventDefault();
                    this.undoLab();
                }
            },

            revertAt(i) {
                this.push();
                this.ops = this.normOps.filter((_, j) => j !== i);
            },

            /** Assigns (wish to Einsatz, or a person from the side panel) without removing anywhere. */
            assign(key, name, shiftId) {
                this.commit(moveOps(Alpine.raw(this.live), this.labModel, key, name, null, shiftId));
            },

            remove(chip, shiftId) {
                this.commit(removeOps(Alpine.raw(this.live), chip.key, chip.name, shiftId));
            },

            /** Moves the dragged or picked person to a shift. */
            dropOn(shiftId) {
                const who = this.dragging || this.picked;
                this.dragging = null;
                this.dragOver = null;
                if (!who) return;
                this.commit(moveOps(Alpine.raw(this.live), this.labModel, who.key, who.name, who.from, shiftId));
                if (this.picked) {
                    this.announce = this.$t('coord.lab_dropped', { name: who.name, shift: this.labShiftLabel(shiftId) });
                    this.picked = null;
                }
            },

            // --- Mouse (native drag and drop) ---

            startDrag(event, who) {
                this.dragging = who;
                event.dataTransfer.effectAllowed = 'move';
                event.dataTransfer.setData('text/plain', who.name); // Firefox needs data to start a drag
            },

            endDrag() {
                this.dragging = null;
                this.dragOver = null;
            },

            overBox(event, shiftId) {
                if (!this.dragging) return;
                event.preventDefault();
                event.dataTransfer.dropEffect = 'move';
                this.dragOver = shiftId;
            },

            leaveBox(event, shiftId) {
                if (this.dragOver === shiftId && !event.currentTarget.contains(event.relatedTarget)) this.dragOver = null;
            },

            // --- Keyboard: pick up and put down ---

            pickUp(who) {
                this.picked = who;
                this.announce = this.$t('coord.lab_picked', { name: who.name });
                this.$nextTick(() => {
                    const target = [...this.$root.querySelectorAll('[data-drop]')].find(el => el.offsetParent !== null);
                    target?.focus();
                });
            },

            chipKey(event, chip, shiftId) {
                const who = { key: chip.key, name: chip.name, from: shiftId };
                switch (event.key) {
                    case 'Enter':
                        event.preventDefault();
                        this.pickUp(who);
                        break;
                    case 'e': case 'E':
                        event.preventDefault();
                        this.assign(chip.key, chip.name, shiftId);
                        break;
                    case 'Delete': case 'Backspace':
                        event.preventDefault();
                        this.remove(chip, shiftId);
                        break;
                }
            },

            // --- People ---

            selectPerson(key) {
                this.selectedKey = this.selectedKey === key ? null : key;
            },

            /** "Hinzufügen": a name only; it becomes a person with the first assignment. */
            addPerson() {
                const name = this.newName.trim().replace(/\s+/g, ' ');
                if (!name) {
                    this.newNameError = this.$t('shift_plans.view.error_name_required');
                    return;
                }
                if ([...name].length > PERSON_NAME_MAX) {
                    this.newNameError = this.$t('shift_plans.view.error_name_too_long', { max: PERSON_NAME_MAX });
                    return;
                }
                this.newNameError = '';
                const key = personKey(name);
                if (!this.deltas.has(key) && !this.extraNames.some(n => personKey(n) === key)) this.extraNames.push(name);
                this.selectedKey = key;
                this.newName = '';
            },
        };
    });
});

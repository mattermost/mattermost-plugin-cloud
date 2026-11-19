import {
    canUpdateInstallation,
    formatDeletionTime,
    getDomain,
    getVersionLabel,
    isDeletionUrgent,
    isFailedState,
    isTransitionalState,
    matchesQuery,
    stateLabel,
    stateTone,
} from './installation_utils';

describe('installation state helpers', () => {
    it.each([
        ['stable', 'success'],
        ['hibernating', 'info'],
        ['deleted', 'default'],
        ['update-failed', 'danger'],
        ['creation-failed', 'danger'],
        ['creation-no-compatible-clusters', 'danger'],
        ['update-in-progress', 'warning'],
        ['creation-requested', 'warning'],
        ['deletion-pending', 'warning'],
    ])('maps %s to %s', (state, tone) => {
        expect(stateTone(state)).toBe(tone);
    });

    it('detects transitional and failed states', () => {
        expect(isTransitionalState('update-in-progress')).toBe(true);
        expect(isTransitionalState('stable')).toBe(false);
        expect(isTransitionalState('update-failed')).toBe(false);
        expect(isTransitionalState('deletion-pending')).toBe(false);
        expect(isTransitionalState('')).toBe(false);
        expect(isFailedState('update-failed')).toBe(true);
    });

    it('formats the label', () => {
        expect(stateLabel('update-in-progress')).toBe('update in progress');
        expect(stateLabel('')).toBe('unknown');
    });
});

describe('installation data helpers', () => {
    it('reads the domain and version', () => {
        expect(getDomain({DNSRecords: [{DomainName: 'a.example.com'}]})).toBe('a.example.com');
        expect(getDomain({DNSRecords: []})).toBe('');
        expect(getDomain({})).toBe('');
        expect(getVersionLabel({Tag: '9.5.0', Version: 'sha'})).toBe('9.5.0');
        expect(getVersionLabel({Tag: '', Version: 'sha'})).toBe('sha');
    });

    it('allows owners and shared updaters to update', () => {
        expect(canUpdateInstallation({OwnerID: 'me'}, 'me')).toBe(true);
        expect(canUpdateInstallation({OwnerID: 'other', Shared: true, AllowSharedUpdates: true}, 'me')).toBe(true);
        expect(canUpdateInstallation({OwnerID: 'other', Shared: true, AllowSharedUpdates: false}, 'me')).toBe(false);
        expect(canUpdateInstallation({OwnerID: 'other', AllowSharedUpdates: true}, 'me')).toBe(false);
        expect(canUpdateInstallation({}, '')).toBe(false);
    });
});

describe('deletion time', () => {
    const now = new Date('2026-01-01T00:00:00Z');
    const at = (minutes) => new Date(now.getTime() + (minutes * 60 * 1000)).toISOString();

    it('formats the remaining time', () => {
        expect(formatDeletionTime(at((2 * 24 * 60) + 120), now)).toBe('2 days and 2 hours');
        expect(formatDeletionTime(at(24 * 60), now)).toBe('1 day');
        expect(formatDeletionTime(at(180), now)).toBe('3 hours');
        expect(formatDeletionTime(at(5), now)).toBe('5 minutes');
        expect(formatDeletionTime(at(0), now)).toBe('less than a minute');
    });

    it('flags deletions within a day as urgent', () => {
        expect(isDeletionUrgent(at(60), now)).toBe(true);
        expect(isDeletionUrgent(at(25 * 60), now)).toBe(false);
    });
});

describe('matchesQuery', () => {
    const installation = {Name: 'Alpha', Tag: '9.5.0', Version: 'sha', DNSRecords: [{DomainName: 'alpha.example.com'}]};

    it('matches name, domain, and version without case', () => {
        expect(matchesQuery(installation, 'ALP')).toBe(true);
        expect(matchesQuery(installation, 'example.com')).toBe(true);
        expect(matchesQuery(installation, '9.5')).toBe(true);
        expect(matchesQuery(installation, 'beta')).toBe(false);
    });

    it('matches everything for an empty query', () => {
        expect(matchesQuery(installation, '')).toBe(true);
        expect(matchesQuery(installation, '   ')).toBe(true);
        expect(matchesQuery({}, null)).toBe(true);
    });
});

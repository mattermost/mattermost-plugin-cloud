const STATE_TONES = {
    stable: 'success',
    hibernating: 'info',
    deleted: 'default',
};

export function isFailedState(state) {
    return typeof state === 'string' && (state.endsWith('-failed') || state === 'creation-no-compatible-clusters');
}

// Pending deletion lasts up to 24 hours and changes only when someone acts, so it is not transitional.
export function isTransitionalState(state) {
    return Boolean(state) && !STATE_TONES[state] && !isFailedState(state) && state !== 'deletion-pending';
}

export function stateTone(state) {
    if (isFailedState(state)) {
        return 'danger';
    }
    return STATE_TONES[state] || 'warning';
}

export function stateLabel(state) {
    return (state || 'unknown').replace(/-/g, ' ');
}

export function getDomain(installation) {
    const records = installation.DNSRecords || [];
    return records.length > 0 ? records[0].DomainName : '';
}

export function getVersionLabel(installation) {
    return installation.Tag || installation.Version || '';
}

export function isOwner(installation, userID) {
    return Boolean(installation.OwnerID) && installation.OwnerID === userID;
}

// Owners and users of a shared installation with updates enabled can change its configuration.
export function canUpdateInstallation(installation, userID) {
    return isOwner(installation, userID) || Boolean(installation.Shared && installation.AllowSharedUpdates);
}

export function isDeleted(installation) {
    return installation.State === 'deleted';
}

export function formatDeletionTime(timestamp, now = new Date()) {
    const diffInMinutes = Math.floor((new Date(timestamp) - now) / (1000 * 60));
    const diffInHours = Math.floor(diffInMinutes / 60);
    const diffInDays = Math.floor(diffInHours / 24);
    const remainingHours = diffInHours % 24;
    const plural = (count, unit) => `${count} ${unit}${count > 1 ? 's' : ''}`;

    if (diffInDays > 0) {
        if (remainingHours > 0) {
            return `${plural(diffInDays, 'day')} and ${plural(remainingHours, 'hour')}`;
        }
        return plural(diffInDays, 'day');
    }
    if (diffInHours > 0) {
        return plural(diffInHours, 'hour');
    }
    if (diffInMinutes > 0) {
        return plural(diffInMinutes, 'minute');
    }
    return 'less than a minute';
}

export function isDeletionUrgent(timestamp, now = new Date()) {
    return (new Date(timestamp) - now) < (1000 * 60 * 60 * 24);
}

// Matches the query against the name, the domain, and the version. An empty query matches all.
export function matchesQuery(installation, query) {
    const needle = (query || '').trim().toLowerCase();
    if (!needle) {
        return true;
    }

    return [installation.Name, getDomain(installation), getVersionLabel(installation)].
        some((value) => (value || '').toLowerCase().includes(needle));
}

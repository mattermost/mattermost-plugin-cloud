import {getCurrentUserId} from 'mattermost-redux/selectors/entities/common';

import {receivedInstallationEvent, refreshInstallations} from './actions';
import {id as pluginId} from './manifest';
import {isRhsVisible} from './selectors';

export const INSTALLATION_UPDATED_EVENT = `custom_${pluginId}_installation_updated`;

// Only a mounted RHS reloads. It loads the lists on mount, so a hidden RHS needs no reload.
// Provisioner webhooks arrive in bursts. One reload covers all events in the window.
const REFRESH_DELAY_MILLISECONDS = 500;

export function createInstallationEventHandler(store, delay = REFRESH_DELAY_MILLISECONDS) {
    let timer = null;
    let pending = {mine: false, shared: false};

    const flush = () => {
        const scope = pending;
        pending = {mine: false, shared: false};
        timer = null;
        if (isRhsVisible(store.getState())) {
            store.dispatch(refreshInstallations(scope));
        }
    };

    return (message) => {
        const data = message && message.data;
        if (!data || !data.installation_id) {
            return;
        }

        store.dispatch(receivedInstallationEvent(data));

        const isOwner = data.owner_id === getCurrentUserId(store.getState());
        pending.mine = pending.mine || isOwner;
        pending.shared = pending.shared || Boolean(data.shared) || !isOwner;

        if (timer === null) {
            timer = setTimeout(flush, delay);
        }
    };
}

export function createReconnectHandler(store) {
    return () => {
        if (isRhsVisible(store.getState())) {
            store.dispatch(refreshInstallations());
        }
    };
}

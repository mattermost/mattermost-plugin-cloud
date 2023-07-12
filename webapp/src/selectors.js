import {getCurrentUserId} from 'mattermost-redux/selectors/entities/common';

import {id as pluginId} from './manifest';

const getPluginState = (state) => state['plugins-' + pluginId] || {};

export const installsForUser = (state, id) => getPluginState(state).cloudUserInstalls[id] || [];
export const getShowRHSAction = (state) => getPluginState(state).rhsPluginAction;

export const isRhsVisible = (state) => getPluginState(state).isRhsVisible;
export const serverError = (state) => getPluginState(state).serverError;
export const deletionLockedInstallId = (state) => {
    const currentUserId = getCurrentUserId(state);
    const installs = getPluginState(state).cloudUserInstalls[currentUserId] || [];
    const installID = Object.keys(installs).find((key) => installs[key].deletion_locked);
    return installID;
};

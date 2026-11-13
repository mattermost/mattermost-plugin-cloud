import {getConfig} from 'mattermost-redux/selectors/entities/general';

import {Client4} from 'mattermost-redux/client';
import {getCurrentChannel} from 'mattermost-redux/selectors/entities/channels';
import {getCurrentTeamId} from 'mattermost-redux/selectors/entities/teams';
import {getCurrentUserId} from 'mattermost-redux/selectors/entities/common';

import Client from '../client';

import {id as pluginId} from '../manifest';

import {rawInstallsEntry, serverError} from '../selectors';

import {
    RECEIVED_USER_INSTALLS,
    RECEIVED_SHARED_INSTALLS,
    RECEIVED_SHOW_RHS_ACTION,
    SET_RHS_VISIBLE,
    SET_SERVER_ERROR,
    RECEIVED_CONFIG,
    RECEIVED_INSTALLATION_EVENT,
} from '../action_types';

const CLOUD_USER_GET_TIMEOUT_MILLISECONDS = 1000 * 60; // 1 minute

// The client throws on HTTP errors. Callers expect {error: {status, message}}.
async function fetchOrError(call) {
    try {
        return await call();
    } catch (error) {
        return {error: {status: error && error.status_code, message: error && error.message}};
    }
}

export function setServerError(errorString) {
    return {
        type: SET_SERVER_ERROR,
        error: errorString,
    };
}

export const deletionLockInstallation = (installationID) => installationMutation(() => Client.deletionLockInstallation(installationID));
export const deletionUnlockInstallation = (installationID) => installationMutation(() => Client.deletionUnlockInstallation(installationID));

export function getPluginConfiguration() {
    return async (dispatch, getState) => {
        const data = await fetchOrError(() => Client.getPluginConfiguration());

        if (data && data.error) {
            dispatch(setServerError(`Status: ${data.error.status}, Message: ${data.error.message}`));
            return data;
        }

        // Clear server error
        if (serverError(getState())) {
            dispatch(setServerError(''));
        }

        dispatch({
            type: RECEIVED_CONFIG,
            data,
        });

        return {data};
    };
}

export function getCloudUserData(userID) {
    return async (dispatch, getState) => {
        if (!userID) {
            return {};
        }

        const installs = rawInstallsEntry(getState(), userID);
        if (installs && installs.last_try && Date.now() - installs.last_try < CLOUD_USER_GET_TIMEOUT_MILLISECONDS) {
            return {};
        }

        const data = await fetchOrError(() => Client.getUserInstalls(userID));
        if (data && data.error) {
            if (data.error.status === 404) {
                dispatch({
                    type: RECEIVED_USER_INSTALLS,
                    userID,
                    data: {last_try: Date.now()},
                });
            }
            dispatch(setServerError(`Status: ${data.error.status}, Message: ${data.error.message}`));
            return data;
        }

        // Clear server error
        if (serverError(getState())) {
            dispatch(setServerError(''));
        }

        dispatch({
            type: RECEIVED_USER_INSTALLS,
            userID,
            data,
        });

        return {data};
    };
}

export function getSharedInstalls() {
    return async (dispatch, getState) => {
        const data = await fetchOrError(() => Client.getSharedInstalls());

        if (data && data.error) {
            dispatch(setServerError(`Status: ${data.error.status}, Message: ${data.error.message}`));
            return data;
        }

        // Clear server error
        if (serverError(getState())) {
            dispatch(setServerError(''));
        }

        dispatch({
            type: RECEIVED_SHARED_INSTALLS,
            data,
        });

        return {data};
    };
}

const errorMessage = (error) => {
    const message = error && error.message ? String(error.message).trim() : '';
    return message || 'Unexpected error';
};

export function refreshInstallations({mine = true, shared = true} = {}) {
    return async (dispatch, getState) => {
        const userID = getCurrentUserId(getState());

        // Background refreshes ignore failures. The next event or the next explicit load retries.
        const tasks = [];
        if (mine && userID) {
            tasks.push(Client.getUserInstalls(userID).then((data) => {
                dispatch({type: RECEIVED_USER_INSTALLS, userID, data});
            }));
        }
        if (shared) {
            tasks.push(Client.getSharedInstalls().then((data) => {
                dispatch({type: RECEIVED_SHARED_INSTALLS, data});
            }));
        }

        await Promise.all(tasks.map((task) => task.catch(() => null)));
    };
}

// Runs a mutation, then reloads the lists. Resolves to {data} or {error: string}.
function installationMutation(call) {
    return async (dispatch) => {
        let data;
        try {
            data = await call();
        } catch (error) {
            return {error: errorMessage(error)};
        }

        await dispatch(refreshInstallations());
        return {data};
    };
}

export const createInstallation = (input) => installationMutation(() => Client.createInstallation(input));
export const updateInstallation = (input) => installationMutation(() => Client.updateInstallation(input));
export const restartInstallation = (installationID) => installationMutation(() => Client.restartInstallation(installationID));
export const hibernateInstallation = (installationID) => installationMutation(() => Client.hibernateInstallation(installationID));
export const wakeInstallation = (installationID) => installationMutation(() => Client.wakeInstallation(installationID));
export const deleteInstallation = (installationID, confirmName) => installationMutation(() => Client.deleteInstallation(installationID, confirmName));
export const shareInstallation = (installationID, shared, allowUpdates) => installationMutation(() => Client.shareInstallation(installationID, shared, allowUpdates));

// Patches the state of a known installation right away. Callers reload the lists afterwards.
export function receivedInstallationEvent(data) {
    return {
        type: RECEIVED_INSTALLATION_EVENT,
        event: {
            installationID: data.installation_id,
            ownerID: data.owner_id,
            state: data.state,
            shared: Boolean(data.shared),
        },
    };
}

export function setShowRHSAction(showRHSPluginAction) {
    return {
        type: RECEIVED_SHOW_RHS_ACTION,
        showRHSPluginAction,
    };
}

export function setRhsVisible(payload) {
    return {
        type: SET_RHS_VISIBLE,
        payload,
    };
}

export const getPluginServerRoute = (state) => {
    const config = getConfig(state);

    let basePath = '';
    if (config && config.SiteURL) {
        basePath = new URL(config.SiteURL).pathname;

        if (basePath && basePath[basePath.length - 1] === '/') {
            basePath = basePath.substr(0, basePath.length - 1);
        }
    }

    return basePath + '/plugins/' + pluginId;
};

export function executeCommand(command) {
    return async (dispatch, getState) => {
        const currentChannelID = getCurrentChannel(getState()).id;
        const currentTeamID = getCurrentTeamId(getState());

        await Client.clientExecuteCommand(command, currentChannelID, currentTeamID);

        return {data: null};
    };
}

export function getDebugPacket(name) {
    return executeCommand(`/cloud debug-packet ${name}`);
}

export const telemetry = (event, properties) => async (dispatch, getState) => {
    await fetch(getPluginServerRoute(getState()) + '/telemetry', Client4.getOptions({
        method: 'post',
        body: JSON.stringify({event, properties}),
    }));
};

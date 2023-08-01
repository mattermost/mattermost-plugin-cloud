import {getConfig} from 'mattermost-redux/selectors/entities/general';

import {Client4} from 'mattermost-redux/client';

import Client from '../client';

import {id as pluginId} from '../manifest';

import {installsForUser, serverError} from 'selectors';

import {
    RECEIVED_USER_INSTALLS,
    RECEIVED_SHOW_RHS_ACTION,
    SET_RHS_VISIBLE,
    SET_SERVER_ERROR,
    RECEIVED_CONFIG,
} from '../action_types';

const CLOUD_USER_GET_TIMEOUT_MILLISECONDS = 1000 * 60; // 1 minute

export function setServerError(errorString) {
    return {
        type: SET_SERVER_ERROR,
        error: errorString,
    };
}

export function deletionLockInstallation(installationID) {
    return async (dispatch, getState) => {
        const data = await Client.deletionLockInstallation(installationID);

        if (data.error) {
            dispatch(setServerError(`Status: ${data.error.status}, Message: ${data.error.message}`));
            return data;
        }

        // Clear server error
        if (serverError(getState())) {
            dispatch(setServerError(''));
        }

        return {data};
    };
}

export function deletionUnlockInstallation(installationID) {
    return async (dispatch, getState) => {
        const data = await Client.deletionUnlockInstallation(installationID);

        if (data.error) {
            dispatch(setServerError(`Status: ${data.error.status}, Message: ${data.error.message}`));
            return data;
        }

        // Clear server error
        if (serverError(getState())) {
            dispatch(setServerError(''));
        }

        return {data};
    };
}

export function getPluginConfiguration() {
    return async (dispatch, getState) => {
        const data = await Client.getPluginConfiguration();

        if (data.error) {
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

        const installs = installsForUser(getState(), userID);
        if (installs && installs.last_try && Date.now() - installs.last_try < CLOUD_USER_GET_TIMEOUT_MILLISECONDS) {
            return {};
        }

        const data = await Client.getUserInstalls(userID);

        if (data.error) {
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
export const telemetry = (event, properties) => async (dispatch, getState) => {
    await fetch(getPluginServerRoute(getState()) + '/telemetry', Client4.getOptions({
        method: 'post',
        body: JSON.stringify({event, properties}),
    }));
};

import {getConfig} from 'mattermost-redux/selectors/entities/general';

import {Client4} from 'mattermost-redux/client';

import Client from '../client';

import {id as pluginId} from '../manifest';

import {installsForUser} from 'selectors';

import {
    RECEIVED_USER_INSTALLS,
    RECEIVED_SHOW_RHS_ACTION,
    UPDATE_RHS_STATE,
    SET_RHS_VISIBLE,
} from '../action_types';

const CLOUD_USER_GET_TIMEOUT_MILLISECONDS = 1000 * 60; // 1 minute

export function getCloudUserData(userID) {
    return async (dispatch, getState) => {
        if (!userID) {
            return {};
        }

        const installs = installsForUser(getState(), userID);
        if (installs && installs.last_try && Date.now() - installs.last_try < CLOUD_USER_GET_TIMEOUT_MILLISECONDS) {
            return {};
        }

        let data;
        try {
            data = await Client.getUserInstalls(userID);
        } catch (error) {
            if (error.status === 404) {
                dispatch({
                    type: RECEIVED_USER_INSTALLS,
                    userID,
                    data: {last_try: Date.now()},
                });
            }
            return {error};
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

export function updateRhsState(rhsState) {
    return {
        type: UPDATE_RHS_STATE,
        state: rhsState,
    };
}
export function addInstall(name){
    return async (dispatch, getState) => {
        if (!name) {
            return {};
        }
        const command = `/cloud create ${name}`;
        await Client.clientExecuteCommand(getState, command);

        return {data: null};
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

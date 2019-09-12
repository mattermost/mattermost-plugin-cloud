import Client from '../client';
import ActionTypes from '../action_types';

import {installsForUser} from 'selectors';

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
                    type: ActionTypes.RECEIVED_USER_INSTALLS,
                    userID,
                    data: {last_try: Date.now()},
                });
            }
            return {error};
        }

        dispatch({
            type: ActionTypes.RECEIVED_USER_INSTALLS,
            userID,
            data,
        });

        return {data};
    };
}

import {combineReducers} from 'redux';

import {
    RECEIVED_USER_INSTALLS,
    RECEIVED_SHARED_INSTALLS,
    RECEIVED_SHOW_RHS_ACTION,
    SET_RHS_VISIBLE,
    SET_SERVER_ERROR,
    RECEIVED_CONFIG,
    RECEIVED_INSTALLATION_EVENT,
    SET_SHARED_ERROR,
} from '../action_types';

function applyInstallationState(installs, event) {
    if (!Array.isArray(installs)) {
        return installs;
    }

    let changed = false;
    const next = installs.map((install) => {
        if (install.ID !== event.installationID || install.State === event.state) {
            return install;
        }
        changed = true;
        return {...install, State: event.state};
    });

    return changed ? next : installs;
}

function cloudUserInstalls(state = {}, action) {
    switch (action.type) {
    case RECEIVED_USER_INSTALLS: {
        const nextState = {...state};
        nextState[action.userID] = action.data;
        return nextState;
    }
    case RECEIVED_INSTALLATION_EVENT: {
        const nextState = {};
        let changed = false;
        Object.keys(state).forEach((userID) => {
            nextState[userID] = applyInstallationState(state[userID], action.event);
            changed = changed || nextState[userID] !== state[userID];
        });
        return changed ? nextState : state;
    }
    default:
        return state;
    }
}

function sharedInstalls(state = [], action) {
    switch (action.type) {
    case RECEIVED_SHARED_INSTALLS:
        return action.data;
    case RECEIVED_INSTALLATION_EVENT:
        return applyInstallationState(state, action.event);
    default:
        return state;
    }
}

function sharedError(state = '', action) {
    switch (action.type) {
    case SET_SHARED_ERROR:
        return action.error;
    case RECEIVED_SHARED_INSTALLS:
        return '';
    default:
        return state;
    }
}

function pluginConfiguration(state = {}, action) {
    switch (action.type) {
    case RECEIVED_CONFIG:
        return action.data;
    default:
        return state;
    }
}

function rhsPluginAction(state = null, action) {
    switch (action.type) {
    case RECEIVED_SHOW_RHS_ACTION:
        return action.showRHSPluginAction;
    default:
        return state;
    }
}

function isRhsVisible(state = false, action) {
    switch (action.type) {
    case SET_RHS_VISIBLE:
        return action.payload;
    default:
        return state;
    }
}

function serverError(state = '', action) {
    switch (action.type) {
    case SET_SERVER_ERROR:
        return action.error;
    default:
        return state;
    }
}

export default combineReducers({
    cloudUserInstalls,
    sharedInstalls,
    sharedError,
    rhsPluginAction,
    isRhsVisible,
    serverError,
    pluginConfiguration,
});

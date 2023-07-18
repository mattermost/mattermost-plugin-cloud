import {combineReducers} from 'redux';

import {RECEIVED_USER_INSTALLS, RECEIVED_SHOW_RHS_ACTION, SET_RHS_VISIBLE, SET_SERVER_ERROR, RECEIVED_CONFIG} from '../action_types';

function cloudUserInstalls(state = {}, action) {
    switch (action.type) {
    case RECEIVED_USER_INSTALLS: {
        const nextState = {...state};
        nextState[action.userID] = action.data;
        return nextState;
    }
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
    rhsPluginAction,
    isRhsVisible,
    serverError,
    pluginConfiguration,
});

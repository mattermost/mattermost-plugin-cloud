import {combineReducers} from 'redux';

import {RECEIVED_USER_INSTALLS, RECEIVED_SHOW_RHS_ACTION, UPDATE_RHS_STATE, SET_RHS_VISIBLE} from '../action_types';

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
function rhsPluginAction(state = null, action) {
    switch (action.type) {
    case RECEIVED_SHOW_RHS_ACTION:
        return action.showRHSPluginAction;
    default:
        return state;
    }
}

function rhsState(state = '', action) {
    switch (action.type) {
    case UPDATE_RHS_STATE:
        return action.state;
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

export default combineReducers({
    cloudUserInstalls,
    rhsPluginAction,
    rhsState,
    isRhsVisible,
});

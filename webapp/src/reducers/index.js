import {combineReducers} from 'redux';

import {OPEN_ROOT_MODAL, CLOSE_ROOT_MODAL, RECEIVED_USER_INSTALLS, RECEIVED_SHOW_RHS_ACTION, UPDATE_RHS_STATE, SET_RHS_VISIBLE} from '../action_types';

const rootModalVisible = (state = false, action) => {
    console.log(action.type);
    switch (action.type) {
    case OPEN_ROOT_MODAL:
        return true;
    case CLOSE_ROOT_MODAL:
        return false;
    default:
        return state;
    }
};
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
    rootModalVisible,
    cloudUserInstalls,
    rhsPluginAction,
    rhsState,
    isRhsVisible,
});

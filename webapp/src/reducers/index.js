import {combineReducers} from 'redux';

import ActionTypes from '../action_types';

function cloudUserInstalls(state = {}, action) {
    switch (action.type) {
    case ActionTypes.RECEIVED_USER_INSTALLS: {
        const nextState = {...state};
        nextState[action.userID] = action.data;
        return nextState;
    }
    default:
        return state;
    }
}

export default combineReducers({
    cloudUserInstalls,
});

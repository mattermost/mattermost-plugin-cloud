import {RECEIVED_INSTALLATION_EVENT, RECEIVED_SHARED_INSTALLS, RECEIVED_USER_INSTALLS} from '../action_types';

import reducer from './index';

const event = (overrides) => ({type: RECEIVED_INSTALLATION_EVENT, event: {installationID: 'a', state: 'stable', ...overrides}});

function seed() {
    let state = reducer({}, {type: 'init'});
    state = reducer(state, {type: RECEIVED_USER_INSTALLS, userID: 'me', data: [{ID: 'a', State: 'update-in-progress'}, {ID: 'b', State: 'stable'}]});
    state = reducer(state, {type: RECEIVED_SHARED_INSTALLS, data: [{ID: 'a', State: 'update-in-progress'}, {ID: 'c', State: 'stable'}]});
    return state;
}

describe('installation event reducer', () => {
    it('patches the state of a matching installation in both lists', () => {
        const state = reducer(seed(), event());

        expect(state.cloudUserInstalls.me).toEqual([{ID: 'a', State: 'stable'}, {ID: 'b', State: 'stable'}]);
        expect(state.sharedInstalls).toEqual([{ID: 'a', State: 'stable'}, {ID: 'c', State: 'stable'}]);
    });

    it('keeps references when nothing changes', () => {
        const before = seed();
        const after = reducer(before, event({installationID: 'unknown'}));

        expect(after.cloudUserInstalls).toBe(before.cloudUserInstalls);
        expect(after.sharedInstalls).toBe(before.sharedInstalls);
    });

    it('ignores non-list entries such as the last_try marker', () => {
        let state = seed();
        state = reducer(state, {type: RECEIVED_USER_INSTALLS, userID: 'broken', data: {last_try: 1}});

        expect(() => reducer(state, event())).not.toThrow();
        expect(reducer(state, event()).cloudUserInstalls.broken).toEqual({last_try: 1});
    });
});

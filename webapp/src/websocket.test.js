import {createInstallationEventHandler, createReconnectHandler, INSTALLATION_UPDATED_EVENT} from './websocket';
import {refreshInstallations} from './actions';
import {RECEIVED_INSTALLATION_EVENT} from './action_types';

jest.mock('mattermost-redux/client', () => ({Client4: {}}));
jest.mock('mattermost-redux/selectors/entities/channels', () => ({getCurrentChannel: jest.fn()}));
jest.mock('mattermost-redux/selectors/entities/teams', () => ({getCurrentTeamId: jest.fn()}));
jest.mock('mattermost-redux/selectors/entities/general', () => ({getConfig: jest.fn()}));

jest.mock('./actions', () => ({
    receivedInstallationEvent: jest.requireActual('./actions').receivedInstallationEvent,
    refreshInstallations: jest.fn((scope) => ({type: 'refresh', scope})),
}));

jest.mock('mattermost-redux/selectors/entities/common', () => ({
    getCurrentUserId: () => 'me',
}));

describe('installation websocket handler', () => {
    let store;
    let visible;

    beforeEach(() => {
        jest.useFakeTimers();
        refreshInstallations.mockClear();
        visible = true;
        store = {dispatch: jest.fn(), getState: () => ({'plugins-com.mattermost.cloud': {isRhsVisible: visible}})};
    });

    afterEach(() => {
        jest.useRealTimers();
    });

    const message = (data) => ({event: INSTALLATION_UPDATED_EVENT, data});

    it('listens for the plugin custom event', () => {
        expect(INSTALLATION_UPDATED_EVENT).toBe('custom_com.mattermost.cloud_installation_updated');
    });

    it('patches the state right away and reloads after a short delay', () => {
        const handle = createInstallationEventHandler(store, 100);

        handle(message({installation_id: 'a', owner_id: 'me', state: 'stable', shared: false}));

        expect(store.dispatch).toHaveBeenCalledWith({
            type: RECEIVED_INSTALLATION_EVENT,
            event: {installationID: 'a', ownerID: 'me', state: 'stable', shared: false},
        });
        expect(refreshInstallations).not.toHaveBeenCalled();

        jest.advanceTimersByTime(100);

        expect(refreshInstallations).toHaveBeenCalledWith({mine: true, shared: false});
    });

    it('reloads only the shared list for another user event', () => {
        const handle = createInstallationEventHandler(store, 100);

        handle(message({installation_id: 'a', owner_id: 'other', state: 'stable', shared: true}));
        jest.advanceTimersByTime(100);

        expect(refreshInstallations).toHaveBeenCalledWith({mine: false, shared: true});
    });

    it('coalesces a burst of events into one reload', () => {
        const handle = createInstallationEventHandler(store, 100);

        handle(message({installation_id: 'a', owner_id: 'me', state: 'update-requested'}));
        handle(message({installation_id: 'a', owner_id: 'me', state: 'update-in-progress'}));
        handle(message({installation_id: 'b', owner_id: 'other', state: 'stable', shared: true}));
        jest.advanceTimersByTime(100);

        expect(refreshInstallations).toHaveBeenCalledTimes(1);
        expect(refreshInstallations).toHaveBeenCalledWith({mine: true, shared: true});
    });

    it('ignores malformed messages', () => {
        const handle = createInstallationEventHandler(store, 100);

        handle(null);
        handle(message({}));
        jest.advanceTimersByTime(100);

        expect(store.dispatch).not.toHaveBeenCalled();
    });

    it('skips the reload while the RHS is hidden', () => {
        visible = false;
        const handle = createInstallationEventHandler(store, 100);

        handle(message({installation_id: 'a', owner_id: 'me', state: 'stable'}));
        jest.advanceTimersByTime(100);
        createReconnectHandler(store)();

        expect(refreshInstallations).not.toHaveBeenCalled();
        expect(store.dispatch).toHaveBeenCalledTimes(1);
    });

    it('reloads everything after a reconnect', () => {
        createReconnectHandler(store)();

        expect(refreshInstallations).toHaveBeenCalledWith();
    });
});

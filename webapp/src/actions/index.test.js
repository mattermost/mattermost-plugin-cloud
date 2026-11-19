import Client from '../client';

import {SET_SHARED_ERROR, SET_SERVER_ERROR, RECEIVED_SHARED_INSTALLS, RECEIVED_USER_INSTALLS} from '../action_types';

import {createInstallation, deleteInstallation, deletionLockInstallation, getCloudUserData, getSharedInstalls, refreshInstallations} from './index';

jest.mock('mattermost-redux/client', () => ({Client4: {}}));
jest.mock('mattermost-redux/selectors/entities/channels', () => ({getCurrentChannel: jest.fn()}));
jest.mock('mattermost-redux/selectors/entities/teams', () => ({getCurrentTeamId: jest.fn()}));
jest.mock('mattermost-redux/selectors/entities/general', () => ({getConfig: jest.fn()}));

jest.mock('../client', () => ({
    __esModule: true,
    default: {
        createInstallation: jest.fn(),
        deleteInstallation: jest.fn(),
        deletionLockInstallation: jest.fn(),
        getUserInstalls: jest.fn(),
        getSharedInstalls: jest.fn(),
    },
}));

jest.mock('mattermost-redux/selectors/entities/common', () => ({
    getCurrentUserId: () => 'me',
}));

describe('installation actions', () => {
    let dispatch;
    const getState = () => ({});

    beforeEach(() => {
        jest.clearAllMocks();
        Client.getUserInstalls.mockResolvedValue([{ID: 'a'}]);
        Client.getSharedInstalls.mockResolvedValue([{ID: 'b'}]);
        dispatch = jest.fn((action) => (typeof action === 'function' ? action(dispatch, getState) : action));
    });

    it('returns the data and reloads both lists on success', async () => {
        Client.createInstallation.mockResolvedValue({status: 'create_requested'});

        const result = await createInstallation({name: 'x'})(dispatch, getState);

        expect(result).toEqual({data: {status: 'create_requested'}});
        expect(Client.createInstallation).toHaveBeenCalledWith({name: 'x'});
        expect(Client.getUserInstalls).toHaveBeenCalledWith('me');
        expect(Client.getSharedInstalls).toHaveBeenCalled();
    });

    it('returns the trimmed server message on failure and does not reload', async () => {
        Client.deleteInstallation.mockRejectedValue({message: 'confirmation name does not match\n'});

        const result = await deleteInstallation('id', 'wrong')(dispatch, getState);

        expect(result).toEqual({error: 'confirmation name does not match'});
        expect(Client.getUserInstalls).not.toHaveBeenCalled();
    });

    it('returns a generic message when the error has none', async () => {
        Client.deletionLockInstallation.mockRejectedValue(new Error(''));

        const result = await deletionLockInstallation('id')(dispatch, getState);

        expect(result).toEqual({error: 'Unexpected error'});
    });

    it('ignores reload failures', async () => {
        Client.getUserInstalls.mockRejectedValue(new Error('down'));

        await expect(refreshInstallations()(dispatch, getState)).resolves.toBeUndefined();
        expect(Client.getSharedInstalls).toHaveBeenCalled();
    });

    it('reloads only the requested lists', async () => {
        await refreshInstallations({mine: false, shared: true})(dispatch, getState);

        expect(Client.getUserInstalls).not.toHaveBeenCalled();
        expect(Client.getSharedInstalls).toHaveBeenCalled();
    });

    describe('getCloudUserData', () => {
        it.each([404, 500, 502])('stores an array with last_try and sets the server error on %i', async (status) => {
            Client.getUserInstalls.mockRejectedValue({status_code: status, message: 'boom'});

            const result = await getCloudUserData('me')(dispatch, getState);

            expect(result).toEqual({error: {status, message: 'boom'}});
            const received = dispatch.mock.calls.map((c) => c[0]).find((a) => a.type === RECEIVED_USER_INSTALLS);
            expect(Array.isArray(received.data)).toBe(true);
            expect(received.data.last_try).toEqual(expect.any(Number));
            expect(dispatch).toHaveBeenCalledWith({type: SET_SERVER_ERROR, error: `Status: ${status}, Message: boom`});
        });
    });

    describe('out-of-order responses', () => {
        const deferred = () => {
            let resolve;
            const promise = new Promise((done) => {
                resolve = done;
            });
            return {promise, resolve};
        };

        it('drops a shared list response that a newer request superseded', async () => {
            const slow = deferred();
            Client.getSharedInstalls.mockReturnValueOnce(slow.promise).mockResolvedValueOnce([{ID: 'new'}]);

            const first = getSharedInstalls()(dispatch, getState);
            await refreshInstallations({mine: false, shared: true})(dispatch, getState);
            slow.resolve([{ID: 'old'}]);
            await first;

            const received = dispatch.mock.calls.map((c) => c[0]).filter((a) => a.type === RECEIVED_SHARED_INSTALLS);
            expect(received).toEqual([{type: RECEIVED_SHARED_INSTALLS, data: [{ID: 'new'}]}]);
        });

        it('drops a user list response that a newer request superseded', async () => {
            const slow = deferred();
            Client.getUserInstalls.mockReturnValueOnce(slow.promise).mockResolvedValueOnce([{ID: 'new'}]);

            const first = getCloudUserData('me')(dispatch, getState);
            await refreshInstallations({mine: true, shared: false})(dispatch, getState);
            slow.resolve([{ID: 'old'}]);
            await first;

            const received = dispatch.mock.calls.map((c) => c[0]).filter((a) => a.type === RECEIVED_USER_INSTALLS);
            expect(received).toEqual([{type: RECEIVED_USER_INSTALLS, userID: 'me', data: [{ID: 'new'}]}]);
        });
    });

    describe('getSharedInstalls', () => {
        it('records a failure as a shared error and leaves the global server error alone', async () => {
            Client.getSharedInstalls.mockRejectedValue({message: 'Not authorized\n', status_code: 401});

            const result = await getSharedInstalls()(dispatch, getState);

            expect(result.error).toBeDefined();
            expect(dispatch).toHaveBeenCalledWith({type: SET_SHARED_ERROR, error: 'Not authorized'});
            expect(dispatch).not.toHaveBeenCalledWith(expect.objectContaining({type: SET_SERVER_ERROR}));
        });

        it('falls back to the status when the error has no message', async () => {
            Client.getSharedInstalls.mockRejectedValue({status_code: 500});

            await getSharedInstalls()(dispatch, getState);

            expect(dispatch).toHaveBeenCalledWith({type: SET_SHARED_ERROR, error: 'Status: 500'});
        });

        it('stores the list on success', async () => {
            const result = await getSharedInstalls()(dispatch, getState);

            expect(result).toEqual({data: [{ID: 'b'}]});
            expect(dispatch).toHaveBeenCalledWith({type: RECEIVED_SHARED_INSTALLS, data: [{ID: 'b'}]});
        });
    });
});

import Client from '../client';

import {getCloudUserData} from './index';

jest.mock('../client');
jest.mock('../manifest', () => ({id: 'com.mattermost.cloud'}));
jest.mock('selectors', () => ({
    installsForUser: jest.fn(() => []),
    serverError: jest.fn(() => ''),
}), {virtual: true});

describe('getCloudUserData', () => {
    it.each([404, 500, 502])('stores last_try and sets the server error when the request throws %i', async (status) => {
        Client.getUserInstalls = jest.fn().mockRejectedValue({status_code: status, message: 'boom'});
        const dispatch = jest.fn();

        const result = await getCloudUserData('user1')(dispatch, () => ({}));

        expect(result).toEqual({error: {status, message: 'boom'}});
        expect(dispatch).toHaveBeenCalledWith(expect.objectContaining({
            userID: 'user1',
        }));
        const received = dispatch.mock.calls.map((c) => c[0]).find((a) => a.userID === 'user1');
        expect(Array.isArray(received.data)).toBe(true);
        expect(received.data.last_try).toEqual(expect.any(Number));
        expect(dispatch).toHaveBeenCalledWith(expect.objectContaining({
            error: `Status: ${status}, Message: boom`,
        }));
    });
});

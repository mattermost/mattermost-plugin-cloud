import {Client4} from 'mattermost-redux/client';
import {ClientError} from 'mattermost-redux/client/client4';

export default class Client {
    constructor() {
        this.url = '/plugins/com.mattermost.cloud/api/v1';
    }

    getUserInstalls = async (userID) => {
        return this.doPost(`${this.url}/userinstalls`, JSON.stringify({user_id: userID}));
    };

    getSharedInstalls = async () => this.doGet(`${this.url}/sharedinstalls`);

    deletionLockInstallation = async (installationID) => this.doPost(`${this.url}/deletion-lock`, JSON.stringify({installation_id: installationID}));

    deletionUnlockInstallation = async (installationID) => this.doPost(`${this.url}/deletion-unlock`, JSON.stringify({installation_id: installationID}));

    getPluginConfiguration = async () => this.doGet(`${this.url}/config`);

    fetchJSON = async (url, options) => {
        const {data} = await this.doFetchWithResponse(url, {
            headers: {
                Accept: 'application/json',
                'Content-Type': 'application/json',
                ...options.headers,
            },
            ...options,
        });
        return data;
    };

    doFetchWithResponse = async (url, options = {}) => {
        const response = await fetch(url, Client4.getOptions(options));

        let data;
        if (response.ok) {
            data = await response.json();

            return {
                response,
                data,
            };
        }

        data = await response.text();

        throw new ClientError(Client4.url, {
            message: data || '',
            status_code: response.status,
            url,
        });
    };

    doGet = async (url, body, headers = {}) => {
        return this.fetchJSON(url, {
            method: 'get',
            headers: {
                'X-Timezone-Offset': new Date().getTimezoneOffset(),
                ...headers,
            },
            body,
        });
    };

    doPost = async (url, body, headers = {}) => {
        return this.fetchJSON(url, {
            method: 'post',
            headers: {
                'X-Timezone-Offset': new Date().getTimezoneOffset(),
                ...headers,
            },
            body,
        });
    };

    doDelete = async (url, body, headers = {}) => {
        return this.fetchJSON(url, {
            method: 'delete',
            headers: {
                'X-Timezone-Offset': new Date().getTimezoneOffset(),
                ...headers,
            },
            body,
        });
    };

    doPut = async (url, body, headers = {}) => {
        return this.fetchJSON(url, {
            method: 'put',
            headers: {
                'X-Timezone-Offset': new Date().getTimezoneOffset(),
                ...headers,
            },
            body,
        });
    };

    clientExecuteCommand = async (command, channelID, teamID) => {
        const args = {
            channel_id: channelID,
            team_id: teamID,
        };

        try {
            await Client4.executeCommand(command, args);
        } catch (error) {
            console.error(error); //eslint-disable-line no-console
        }
    };
}

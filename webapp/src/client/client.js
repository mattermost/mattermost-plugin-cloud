import request from 'superagent';

export default class Client {
    constructor() {
        this.url = '/plugins/com.mattermost.cloud/api/v1';
    }

    getUserInstalls = async () => {
        return this.doGet(`${this.url}/userinstalls`);
    }

    doGet = async (url, body, headers = {}) => {
        headers['X-Requested-With'] = 'XMLHttpRequest';
        headers['X-Timezone-Offset'] = new Date().getTimezoneOffset();

        try {
            const response = await request.
                get(url).
                set(headers).
                accept('application/json');

            return response.body;
        } catch (err) {
            throw err;
        }
    }

    doPost = async (url, body, headers = {}) => {
        headers['X-Requested-With'] = 'XMLHttpRequest';
        headers['X-Timezone-Offset'] = new Date().getTimezoneOffset();

        try {
            const response = await request.
                post(url).
                send(body).
                set(headers).
                type('application/json').
                accept('application/json');

            return response.body;
        } catch (err) {
            throw err;
        }
    }

    doDelete = async (url, body, headers = {}) => {
        headers['X-Requested-With'] = 'XMLHttpRequest';
        headers['X-Timezone-Offset'] = new Date().getTimezoneOffset();

        try {
            const response = await request.
                delete(url).
                send(body).
                set(headers).
                type('application/json').
                accept('application/json');

            return response.body;
        } catch (err) {
            throw err;
        }
    }

    doPut = async (url, body, headers = {}) => {
        headers['X-Requested-With'] = 'XMLHttpRequest';
        headers['X-Timezone-Offset'] = new Date().getTimezoneOffset();

        try {
            const response = await request.
                put(url).
                send(body).
                set(headers).
                type('application/json').
                accept('application/json');

            return response.body;
        } catch (err) {
            throw err;
        }
    }
}

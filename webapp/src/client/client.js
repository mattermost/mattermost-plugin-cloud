import request from 'superagent';

export default class Client {
    constructor() {
        this.url = '/plugins/com.mattermost.cloud/api/v1';
    }

    getUserInstalls = async (userID) => {
        return this.doPost(`${this.url}/userinstalls`, {user_id: userID});
    }

    doGet = async (url, body, headers = {}) => {
        headers['X-Requested-With'] = 'XMLHttpRequest';
        headers['X-Timezone-Offset'] = new Date().getTimezoneOffset();
        let response;
        try {
            response = await request.
                get(url).
                set(headers).
                accept('application/json');
        } catch (error) {
            return {error};
        }

        return response.body;
    }

    doPost = async (url, body, headers = {}) => {
        headers['X-Requested-With'] = 'XMLHttpRequest';
        headers['X-Timezone-Offset'] = new Date().getTimezoneOffset();
        let response;
        try {
            response = await request.
                post(url).
                send(body).
                set(headers).
                type('application/json').
                accept('application/json');
        } catch (error) {
            return {error};
        }

        return response.body;
    }

    doDelete = async (url, body, headers = {}) => {
        headers['X-Requested-With'] = 'XMLHttpRequest';
        headers['X-Timezone-Offset'] = new Date().getTimezoneOffset();
        let response;
        try {
            response = await request.
                delete(url).
                send(body).
                set(headers).
                type('application/json').
                accept('application/json');
        } catch (error) {
            return {error};
        }

        return response.body;
    }

    doPut = async (url, body, headers = {}) => {
        headers['X-Requested-With'] = 'XMLHttpRequest';
        headers['X-Timezone-Offset'] = new Date().getTimezoneOffset();
        let response;
        try {
            response = await request.
                put(url).
                send(body).
                set(headers).
                type('application/json').
                accept('application/json');
        } catch (error) {
            return {error};
        }

        return response.body;
    }
}
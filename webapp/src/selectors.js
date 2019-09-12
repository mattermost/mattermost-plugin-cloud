import {id as pluginId} from './manifest';

const getPluginState = (state) => state['plugins-' + pluginId] || {};

export const installsForUser = (state, id) => getPluginState(state).cloudUserInstalls[id] || [];

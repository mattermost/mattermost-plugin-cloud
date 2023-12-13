import {id as pluginId} from './manifest';

const getPluginState = (state) => state['plugins-' + pluginId] || {};

export const installsForUser = (state, id) => getPluginState(state).cloudUserInstalls[id] || [];
export const sharedInstalls = (state) => getPluginState(state).sharedInstalls || [];
export const pluginConfiguration = (state) => getPluginState(state).pluginConfiguration;

export const getShowRHSAction = (state) => getPluginState(state).rhsPluginAction;
export const isRhsVisible = (state) => getPluginState(state).isRhsVisible;
export const serverError = (state) => getPluginState(state).serverError;

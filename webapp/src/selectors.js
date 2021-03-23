import {id as pluginId} from './manifest';

const getPluginState = (state) => state['plugins-' + pluginId] || {};

export const installsForUser = (state, id) => getPluginState(state).cloudUserInstalls[id] || [];
export const getShowRHSAction = (state) => getPluginState(state).rhsPluginAction;

export const isRhsVisible = (state) => getPluginState(state).isRhsVisible;
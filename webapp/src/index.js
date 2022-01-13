import UserAttribute from './components/user_attribute';
import Reducer from './reducers';
import {id as pluginId} from './manifest';

import {getPluginServerRoute, setShowRHSAction, telemetry} from './actions/index.js';
import ChannelHeaderButton from './components/channel_header_button';
import SidebarRight from './components/sidebar_right';

class Plugin {
    async initialize(registry, store) {
        registry.registerReducer(Reducer);

        registry.registerPopoverUserAttributesComponent(UserAttribute);

        const {toggleRHSPlugin, showRHSPlugin} = registry.registerRightHandSidebarComponent(SidebarRight, 'Cloud Plugin');
        store.dispatch(setShowRHSAction(() => store.dispatch(showRHSPlugin)));
        registry.registerChannelHeaderButtonAction(
            <ChannelHeaderButton/>,
            () => {
                telemetry('channel_header_click');
                store.dispatch(toggleRHSPlugin);
            },
            'Cloud Plugin',
            'Cloud Plugin',
        );

        if (registry.registerAppBarComponent) {
            const iconURL = getPluginServerRoute(store.getState()) + '/public/app-bar-icon.png';
            registry.registerAppBarComponent(
                iconURL,
                () => {
                    telemetry('app_bar_click');
                    store.dispatch(toggleRHSPlugin);
                },
                'Cloud Plugin',
            );
        }
    }
}

global.window.registerPlugin(pluginId, new Plugin());

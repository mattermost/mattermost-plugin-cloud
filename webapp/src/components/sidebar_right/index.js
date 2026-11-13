// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';
import {getCurrentUserId} from 'mattermost-redux/selectors/entities/common';

import {
    setRhsVisible,
    getCloudUserData,
    getSharedInstalls,
    getPluginConfiguration,
    refreshInstallations,
    getDebugPacket,
    createInstallation,
    updateInstallation,
    restartInstallation,
    hibernateInstallation,
    wakeInstallation,
    deleteInstallation,
    shareInstallation,
    deletionLockInstallation,
    deletionUnlockInstallation,
} from '../../actions';

import {installsForUser, sharedInstalls, serverError, pluginConfiguration} from '../../selectors';

import SidebarRight from './sidebar_right.jsx';

function mapStateToProps(state) {
    const id = getCurrentUserId(state);
    const config = pluginConfiguration(state);
    return {
        id,
        installs: installsForUser(state, id),
        sharedInstalls: sharedInstalls(state),
        serverError: serverError(state) || '',
        maxLockedInstallations: parseInt(config.DeletionLockInstallationsAllowedPerPerson, 10),
        options: config.Options,
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
            setVisible: setRhsVisible,
            getCloudUserData,
            getSharedInstalls,
            getPluginConfiguration,
            refreshInstallations,
            getDebugPacket,
            createInstallation,
            updateInstallation,
            restartInstallation,
            hibernateInstallation,
            wakeInstallation,
            deleteInstallation,
            shareInstallation,
            deletionLockInstallation,
            deletionUnlockInstallation,
        }, dispatch),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(SidebarRight);

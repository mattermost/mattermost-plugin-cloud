// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';
import {getCurrentUserId} from 'mattermost-redux/selectors/entities/common';

import {telemetry, setRhsVisible, getCloudUserData, deletionLockInstallation, deletionUnlockInstallation, getPluginConfiguration} from '../../actions';

import {installsForUser, serverError, pluginConfiguration} from '../../selectors';

import SidebarRight from './sidebar_right.jsx';

function mapStateToProps(state) {
    const id = getCurrentUserId(state);
    return {
        id,
        installs: installsForUser(state, id),
        serverError: serverError(state),
        maxLockedInstallations: parseInt(pluginConfiguration(state).DeletionLockInstallationsAllowedPerPerson, 10),
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
            telemetry,
            getCloudUserData,
            setVisible: setRhsVisible,
            deletionLockInstallation,
            deletionUnlockInstallation,
            getPluginConfiguration,
        }, dispatch),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(SidebarRight);

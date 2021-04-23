// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';
import {getCurrentUserId} from 'mattermost-redux/selectors/entities/common';

import {telemetry, setRhsVisible, getCloudUserData, addInstall, openRootModal} from '../../actions';

import {installsForUser} from '../../selectors';

import SidebarRight from './sidebar_right.jsx';

function mapStateToProps(state) {
    const id = getCurrentUserId(state);
    const installs = installsForUser(state, id);
    return {
        id,
        installs,
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
            telemetry,
            getCloudUserData,
            addInstall,
            openRootModal,
            setVisible: setRhsVisible,
        }, dispatch),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(SidebarRight);
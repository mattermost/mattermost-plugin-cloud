// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';
//question
import {telemetry, setRhsVisible} from '../../actions';

import SidebarRight from './sidebar_right.jsx';

function mapStateToProps(state) {
    /*return {
        rhsState: state['plugins-com.mattermost.plugin-cloud'].rhsState,
    };*/
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
            telemetry,
            setVisible: setRhsVisible,
        }, dispatch),
    };
}

export default connect(mapStateToProps,mapDispatchToProps)(SidebarRight);

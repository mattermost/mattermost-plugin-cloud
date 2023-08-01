import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';
import {getCurrentUserId} from 'mattermost-redux/selectors/entities/common';

import {getCloudUserData} from '../../actions';

import {getShowRHSAction, installsForUser} from '../../selectors';

import SidebarLeft from './sidebar_left';

function mapStateToProps(state) {
    const id = getCurrentUserId(state);
    return {
        id,
        installs: installsForUser(state, id),
        showRHS: getShowRHSAction(state),
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
            getCloudUserData,
        }, dispatch),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(SidebarLeft);
import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getCloudUserData} from '../../actions';

import {installsForUser} from 'selectors';

import UserAttribute from './user_attribute.jsx';

function mapStateToProps(state, ownProps) {
    const id = ownProps.user ? ownProps.user.id : '';
    const installs = installsForUser(state, id);

    return {
        id,
        installs,
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
            getCloudUserData,
        }, dispatch),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(UserAttribute);

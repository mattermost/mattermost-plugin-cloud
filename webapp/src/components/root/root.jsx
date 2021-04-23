import React from 'react';
import PropTypes from 'prop-types';
import {makeStyleFromTheme, changeOpacity} from 'mattermost-redux/utils/theme_utils';

import FullScreenModal from '../modals/full_screen_modal.jsx';

import './root.scss';


const PostUtils = window.PostUtils;

export default class Root extends React.Component {
    static propTypes = {
        visible: PropTypes.bool.isRequired,
        close: PropTypes.func.isRequired,
    }
    constructor(props) {
        super(props);

        this.state = {};
    }

    render() {
        console.log(this.props.visible);
        const {visible, close} = this.props;
        if (!visible) {
            return null;
        }
        return (
            <FullScreenModal
            show={visible}
            onClose={close}
        >
            <div>
                <div>
                    <p>hello</p>
                </div>
            </div>
        </FullScreenModal>
        );
    }
}


// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import PropTypes from 'prop-types';
import Scrollbars from 'react-custom-scrollbars';
import './sidebar_right.scss';

export function renderView(props) {
    return (
        <div
            {...props}
            className='scrollbar--view'
        />);
}

export function renderThumbHorizontal(props) {
    return (
        <div
            {...props}
            className='scrollbar--horizontal'
        />);
}

export function renderThumbVertical(props) {
    return (
        <div
            {...props}
            className='scrollbar--vertical'
        />);
}

export default class SidebarRight extends React.PureComponent {
    static propTypes = {
        rhsState: PropTypes.string,
        actions: PropTypes.shape({
            setVisible: PropTypes.func.isRequired,
            telemetry: PropTypes.func.isRequired,
        }).isRequired,
    };

    constructor(props) {
        super(props);

        this.state = {};
    }

    componentDidMount() {
        this.props.actions.setVisible(true);
    }

    componentWillUnmount() {
        this.props.actions.setVisible(false);
    }



    render() {
        return (
            <React.Fragment>
            <Scrollbars
                autoHide={true}
                autoHideTimeout={500}
                autoHideDuration={500}
                renderThumbHorizontal={renderThumbHorizontal}
                renderThumbVertical={renderThumbVertical}
                renderView={renderView}
                className='SidebarRight'
            >
                <div className='header-menu'>
                    <p>hello</p>
                </div>
            </Scrollbars>
        </React.Fragment>
        );
    }
}

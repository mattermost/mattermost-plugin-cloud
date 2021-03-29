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
        id: PropTypes.string.isRequired,
        installs: PropTypes.array.isRequired,
        actions: PropTypes.shape({
            setVisible: PropTypes.func.isRequired,
            telemetry: PropTypes.func.isRequired,
            getCloudUserData: PropTypes.func.isRequired,
        }).isRequired,
    };

    constructor(props) {
        super(props);

        this.state = {};
    }

    componentDidMount() {
        this.props.actions.setVisible(true);
        console.log("this is the id"+ this.props.id);
        this.props.actions.getCloudUserData(this.props.id);
    }

    componentWillUnmount() {
        this.props.actions.setVisible(false);
    }



    render() {

       // const installs = [{ ID: 1, DNS: "indu.dev", Name: "indu" }, { ID: 2, DNS: "bob.dev", Name: "bob" }, { ID: 3, DNS: "test.dev", Name: "test" }]
         const installs = this.props.installs;
         var noInstalls = ""

         if (installs.length == 0) {
             noInstalls = "No installs for user"
             return null;
         }
  
          const entries = installs.map((install) => (
              <li key={install.ID}>
                  <a
                      href={'https://' + install.DNS}
                      target='_blank'
                      rel='noopener noreferrer'
                  >
                      {install.Name}
                  </a>
              </li>
          ));
          

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
                        {noInstalls}
                    </div>

                    <div style={style.container}>
                        <div>{'Cloud Servers'}</div>
                        <ul style={style.ul}>
                            {entries}
                        </ul>
                    </div>
                </Scrollbars>
            </React.Fragment>
        );
    }
}
const style = {
    container: {
        margin: '5px 0',
    },
    ul:{
        listStyleType:"none",
    },
    li:{
        width:"300px",
        border:"1px solid #000",
        padding:"20px",
    }
};

import React from 'react';
import PropTypes from 'prop-types';
import {Scrollbars} from 'react-custom-scrollbars-2';
import {Label} from 'react-bootstrap';

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
        id: PropTypes.string.isRequired,
        installs: PropTypes.array.isRequired,
        serverError: PropTypes.string.isRequired,
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
        this.props.actions.getCloudUserData(this.props.id);
    }

    componentWillUnmount() {
        this.props.actions.setVisible(false);
    }

    render() {
        if (this.props.serverError) {
            return (
                <div style={style.message}>
                    <p>{'Received a server error'}</p>
                    <p>{this.props.serverError}</p>
                    <div style={style.serverIcon}>
                        <i className='fa fa-server fa-4x'/>
                    </div>

                </div>
            );
        }

        const installs = this.props.installs;
        if (installs.length === 0) {
            return (
                <div style={style.message}>
                    <p>{'There are no installations, use the /cloud create command to add an installation.'}</p>
                    <div style={style.serverIcon}>
                        <i className='fa fa-server fa-4x'/>
                    </div>

                </div>
            );
        }

        const entries = installs.map((install) => (
            <li
                style={style.li}
                key={install.ID}
            >
                <div style={style.header}>
                    <div style={style.nameText}><b>{install.Name}</b></div>
                    <span><Label style={install.State === 'stable' ? style.stable : style.inProgress}><b>{install.State}</b></Label></span>
                </div>
                <div style={style.installinfo}>
                    <div>
                        <span style={style.col1}>DNS:</span>
                        {install.DNSRecords.length > 0 ?
                            <span>{install.DNSRecords[0].DomainName}</span> :
                            <span>No URL!</span>
                        }
                    </div>

                    <div>
                        <span style={style.col1}>Image:</span>
                        <span>{install.Image}</span>
                    </div>
                    <div>
                        {install.Tag === '' ?
                            <div>
                                <span style={style.col1}>Version:</span>
                                <span>{install.Version}</span>
                            </div> :
                            <div>
                                <span style={style.col1}>Tag:</span>
                                <span>{install.Tag}</span>
                            </div>
                        }
                    </div>
                    <div>
                        <span style={style.col1}>Database:</span>
                        <span>{install.Database}</span>
                    </div>
                    <div>
                        <span style={style.col1}>Filestore:</span>
                        <span>{install.Filestore}</span>
                    </div>
                    <div>
                        <span style={style.col1}>Size:</span>
                        <span>{install.Size}</span>
                    </div>
                </div>

                <a
                    href={'https://' + install.DNSRecords[0].DomainName}
                    target='_blank'
                    rel='noopener noreferrer'
                >
                    <div>
                        <button
                            className='btn btn-primary btn-block'
                        >{'View Installation'}
                        </button>

                    </div>

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
                    <div style={style.container}>
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
        margin: '0px 0',
    },
    ul: {
        listStyleType: 'none',
        padding: '0px',
        margin: '0px',
    },
    li: {
        padding: '20px',
    },
    col1: {
        width: '80px',
        float: 'left',
    },
    header: {
        display: 'flex',
        marginBottom: '10px',
    },
    installinfo: {
        fontSize: '12px',
        marginBottom: '15px',
    },
    nameText: {
        paddingRight: '10px',
        fontSize: '16px',
    },
    stable: {
        fontSize: '11px',
        color: 'var(--center-channel-bg)',
        backgroundColor: 'var(--online-indicator)',
    },
    inProgress: {
        fontSize: '11px',
        color: 'var(--center-channel-bg)',
        backgroundColor: 'var(--dnd-indicator)',
    },
    message: {
        margin: 'auto',
        width: '50%',
        marginTop: '50px',
    },
    serverIcon: {
        margin: '0 auto',
        width: '50%',
    },
};


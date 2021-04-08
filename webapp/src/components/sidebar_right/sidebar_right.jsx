import React from 'react';
import PropTypes from 'prop-types';
import Scrollbars from 'react-custom-scrollbars';
import {Badge} from 'react-bootstrap';

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
        this.props.actions.getCloudUserData(this.props.id);
    }

    componentWillUnmount() {
        this.props.actions.setVisible(false);
    }

    render() {
        const installs = this.props.installs;
        var noInstalls = false;

        if (installs.length === 0) {
            noInstalls = true;
        }

        const entries = installs.map((install) => (
            <li
                style={style.li}
                key={install.ID}>
                <div style={style.name}
            >
                    <span><b style={style.nameText}>{install.Name}</b></span>
                </div>
                <div style={style.installinfo}>
                    {install.State === 'stable' ? <div>
                        <span style={style.col1}>State:</span>
                        <span><Badge style={style.successBadge}>{install.State}</Badge></span>
                    </div> :
                        <div>
                            <span style={style.col1}>State:</span>
                            <span><Badge style={style.warningBadge}>{install.State}</Badge></span>
                        </div>

                    }
                    <div>
                        <span style={style.col1}>DNS:</span>
                        <span>{install.DNS}</span>
                    </div>
                    <div>
                        <span style={style.col1}>Image:</span>
                        <span>{install.Image}</span>
                    </div>
                    <div>
                        {install.Tag === "" ?
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
                    href={'https://' + install.DNS}
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
                    {noInstalls ?
                        <div style={style.noInstalls}>

                            <p>There are no installations, use the '/cloud create' command to add an installation.</p>

                            <div style={style.serverIcon}>
                                <i className="fa fa-server fa-4x" />
                            </div>

                        </div> :
                        <div style={style.container}>
                            <ul style={style.ul}>
                                {entries}
                            </ul>
                        </div>}
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
        listStyleType: "none",
        padding: "0px",
        margin:'0px',
    },
    li: {
        borderTop: "1px solid #D3D3D3",
        padding: "20px",
    },
    col1: {
        width: "100px",
        float: 'left',
    },
    name: {
        marginBottom: "5px",
    },
    installinfo: {
        marginBottom: "15px",
    },
    nameText: {
        fontSize: "15px",
    },
    successBadge: {
        width: '50px',
        display: 'inline',
        backgroundColor: '#00FF7F',
    },
    warningBadge: {
        width: '50px',
        display: 'inline',
        backgroundColor: '#FF8C00',
    },
    noInstalls: {
        margin: 'auto',
        width: '50%',
        marginTop: '50px',
    },
    serverIcon: {
        margin: '0 auto',
        width: '50%',
    }

};


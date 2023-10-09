import React from 'react';
import PropTypes from 'prop-types';
import {Scrollbars} from 'react-custom-scrollbars-2';
import {Button, Label, DropdownButton, MenuItem} from 'react-bootstrap';

export function renderView(props) {
    return (
        <div
            {...props}
            className='scrollbar--view'
        />
    );
}

export function renderThumbHorizontal(props) {
    return (
        <div
            {...props}
            className='scrollbar--horizontal'
        />
    );
}

export function renderThumbVertical(props) {
    return (
        <div
            {...props}
            className='scrollbar--vertical'
        />
    );
}

export default class SidebarRight extends React.PureComponent {
    static propTypes = {
        id: PropTypes.string.isRequired,
        installs: PropTypes.array.isRequired,
        serverError: PropTypes.string.isRequired,
        deletionLockedInstallationId: PropTypes.string,
        maxLockedInstallations: PropTypes.number,
        actions: PropTypes.shape({
            setVisible: PropTypes.func.isRequired,
            telemetry: PropTypes.func.isRequired,
            getCloudUserData: PropTypes.func.isRequired,
            deletionLockInstallation: PropTypes.func.isRequired,
            deletionUnlockInstallation: PropTypes.func.isRequired,
            getPluginConfiguration: PropTypes.func.isRequired,
        }).isRequired,
    };

    constructor(props) {
        super(props);

        this.state = {
            deletionLockedInstallationId: null,
        };
    }

    componentDidMount() {
        this.props.actions.setVisible(true);
        this.props.actions.getCloudUserData(this.props.id);
        this.props.actions.getPluginConfiguration();
    }

    componentWillUnmount() {
        this.props.actions.setVisible(false);
    }

    installationButtons(installation) {
        const dropdownButtonItems = [
            {onClick: () => window.open(installation.InstallationLogsURL, '_blank'), buttonText: 'Installation Logs'},
            {onClick: () => window.open(installation.ProvisionerLogsURL, '_blank'), buttonText: 'Provisioner Logs'},
        ];
        const menuItems = dropdownButtonItems.map((menuItem, index) => (
            <MenuItem
                key={'log-menu-' + installation.ID + '-' + index}
                onClick={menuItem.onClick}
            >{menuItem.buttonText}
            </MenuItem>
        ));

        return (
            <div>
                <Button
                    className='btn btn-primary btn-sm'
                    onClick={() => window.open('https://' + installation.DNSRecords[0].DomainName, '_blank')}
                >{'Open'}
                </Button>
                <DropdownButton
                    style={style.dropdownButton}
                    className='btn btn-tertiary btn-sm'
                    title='Logs'
                >
                    {menuItems}
                </DropdownButton>
                {this.deletionLockButton(installation)}
            </div>
        );
    }

    deletionLockButton(installation) {
        const deletionLockedInstallationsIds = this.props.installs.filter((install) => install.DeletionLocked).map((install) => install.ID);
        if (deletionLockedInstallationsIds.includes(installation.ID)) {
            return (
                <Button
                    className='btn btn-danger btn-sm'
                    onClick={async () => {
                        await this.props.actions.deletionUnlockInstallation(installation.ID);
                        this.props.actions.getCloudUserData(this.props.id);
                    }
                    }
                >{'Unlock Deletion'}
                </Button>
            );
        }

        return (
            <Button
                className='btn btn-tertiary btn-sm'
                disabled={deletionLockedInstallationsIds.length >= this.props.maxLockedInstallations}
                onClick={async () => {
                    await this.props.actions.deletionLockInstallation(installation.ID);
                    this.props.actions.getCloudUserData(this.props.id);
                }
                }
            >{'Lock Deletion'}
            </Button>
        );
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
                    <span>
                        <Label style={install.State === 'stable' ? style.stable : style.inProgress}>
                            <b>{install.State}</b>
                        </Label>
                        {install.DeletionLocked &&
                            <Label style={style.stable}>
                                <b>deletion locked</b>
                            </Label>
                        }
                    </span>
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
                {this.installationButtons(install)}
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
        marginRight: '10px',
    },
    inProgress: {
        fontSize: '11px',
        color: 'var(--center-channel-bg)',
        backgroundColor: 'var(--dnd-indicator)',
        marginRight: '10px',
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
    dropdownButton: {
        margin: '0 8px',
    },
};


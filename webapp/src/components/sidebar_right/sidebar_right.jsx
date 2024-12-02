import React from 'react';
import PropTypes from 'prop-types';
import {Scrollbars} from 'react-custom-scrollbars-2';
import {Button, Label, DropdownButton, MenuItem} from 'react-bootstrap';

import ConfirmationModal from './confirmation_modal';

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
        sharedInstalls: PropTypes.array.isRequired,
        serverError: PropTypes.string.isRequired,
        deletionLockedInstallationId: PropTypes.string,
        maxLockedInstallations: PropTypes.number,
        serverTypeValue: PropTypes.string,
        actions: PropTypes.shape({
            setVisible: PropTypes.func.isRequired,
            telemetry: PropTypes.func.isRequired,
            getCloudUserData: PropTypes.func.isRequired,
            getSharedInstalls: PropTypes.func.isRequired,
            restartInstallation: PropTypes.func.isRequired,
            getDebugPacket: PropTypes.func.isRequired,
            deletionLockInstallation: PropTypes.func.isRequired,
            deletionUnlockInstallation: PropTypes.func.isRequired,
            getPluginConfiguration: PropTypes.func.isRequired,
        }).isRequired,
    };

    constructor(props) {
        super(props);

        this.state = {
            deletionLockedInstallationId: null,
            confirmationModal: {visible: false},
            serverTypeValue: 'Personal',
        };
    }

    componentDidMount() {
        this.props.actions.setVisible(true);
        this.props.actions.getCloudUserData(this.props.id);
        this.props.actions.getSharedInstalls();
        this.props.actions.getPluginConfiguration();
    }

    componentWillUnmount() {
        this.props.actions.setVisible(false);
    }

    setServerTypeValue(value) {
        this.setState({serverTypeValue: value});
    }

    installationButtons(installation, shared) {
        const dropdownButtonItems = [
            {onClick: () => window.open(installation.InstallationLogsURL, '_blank'), buttonText: 'Installation Logs'},
            {onClick: () => window.open(installation.ProvisionerLogsURL, '_blank'), buttonText: 'Provisioner Logs'},
            {onClick: () => this.handleGetDebugPacket(installation.Name), buttonText: 'Get Debug Packet'},
        ];
        const menuItems = dropdownButtonItems.map((menuItem, index) => (
            <MenuItem
                key={'debug-menu-' + installation.ID + '-' + index}
                onClick={menuItem.onClick}
            >{menuItem.buttonText}
            </MenuItem>
        ));
        let actionButtons;
        if (!shared) {
            actionButtons = [
                <Button
                    data-testid={'restart-' + installation.ID}
                    key={'restart-' + installation.ID}
                    className='btn btn-tertiary btn-sm'
                    onClick={() => this.handleRestartButtonClick(installation)}
                >{'Restart'}
                </Button>,
                this.deletionLockButton(installation),
            ];
        }

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
                    title='Debug'
                >
                    {menuItems}
                </DropdownButton>
                {actionButtons}
            </div>
        );
    }

    handleRestart = async (installation) => {
        await this.props.actions.restartInstallation(installation.Name);
        this.props.actions.getCloudUserData(this.props.id);
        this.setState({confirmationModal: {visible: false}});
    };

    handleRestartButtonClick(installation) {
        this.setState({confirmationModal: {
            title: 'Restart installation servers?',
            bodyText: 'Are you sure you want to restart the mattermost server instances? Doing so force new servers to be created while removing the old ones.',
            visible: true,
            onConfirm: () => this.handleRestart(installation),
            onCancel: () => this.setState({confirmationModal: {visible: false}}),
        }});
    }

    handleGetDebugPacket = async (name) => {
        await this.props.actions.getDebugPacket(name);
    };

    async handleDeletionUnlock(installation) {
        await this.props.actions.deletionUnlockInstallation(installation.ID);
        this.props.actions.getCloudUserData(this.props.id);
        this.setState({confirmationModal: {visible: false}});
    }

    handleDeletionUnlockButtonClick(installation) {
        this.setState({confirmationModal: {
            title: 'Remove deletion lock?',
            bodyText: 'Are you sure you want to remove the deletion lock? Doing so will add this installation back into the clean up pool, meaning it can be deleted.',
            visible: true,
            onConfirm: () => this.handleDeletionUnlock(installation),
            onCancel: () => this.setState({confirmationModal: {visible: false}}),
        }});
    }

    deletionLockButton(installation) {
        const deletionLockedInstallationsIds = this.props.installs.filter((install) => install.DeletionLocked).map((install) => install.ID);
        if (deletionLockedInstallationsIds.includes(installation.ID)) {
            return (
                <Button
                    className='btn btn-danger btn-sm'
                    onClick={() => this.handleDeletionUnlockButtonClick(installation)}
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

    personalInstallationList(installations) {
        return this.installationList(installations, false);
    }

    sharedInstallationList(installations) {
        return this.installationList(installations, true);
    }

    installationList(installations, shared) {
        return installations.map((installation) => (
            <li
                style={style.li}
                key={installation.ID}
            >
                <div>
                    <div style={style.nameText}><b>{installation.Name}</b></div>
                </div>
                <div style={style.header}>
                    <Label style={installation.State === 'stable' ? style.stable : style.inProgress}>
                        <b>{installation.State}</b>
                        <span
                            style={style.badgeIcon}
                            className={installation.State === 'stable' ? 'fa fa-check' : 'fa fa-spinner'}
                        />
                    </Label>
                    {installation.DeletionLocked &&
                    <Label style={style.stable}>
                        <b>deletion locked</b>
                        <span
                            style={style.badgeIcon}
                            className='fa fa-lock'
                        />
                    </Label>
                    }
                    {installation.Shared &&
                    <Label style={style.stable}>
                        {installation.AllowSharedUpdates ? 'shared-updates' : 'shared'}
                        <span
                            style={style.badgeIcon}
                            className='fa fa-users'
                        />
                    </Label>
                    }
                </div>
                <div style={style.installinfo}>
                    <div>
                        <span style={style.col1}>DNS:</span>
                        {installation.DNSRecords.length > 0 ? <span>{installation.DNSRecords[0].DomainName}</span> : <span>No URL!</span>
                        }
                    </div>

                    <div>
                        <span style={style.col1}>Image:</span>
                        <span>{installation.Image}</span>
                    </div>
                    <div>
                        {installation.Tag === '' ? <div>
                            <span style={style.col1}>Version:</span>
                            <span>{installation.Version}</span>
                        </div> : <div>
                            <span style={style.col1}>Tag:</span>
                            <span>{installation.Tag}</span>
                        </div>
                        }
                    </div>
                    <div>
                        <span style={style.col1}>Database:</span>
                        <span>{installation.Database}</span>
                    </div>
                    <div>
                        <span style={style.col1}>Filestore:</span>
                        <span>{installation.Filestore}</span>
                    </div>
                    <div>
                        <span style={style.col1}>Size:</span>
                        <span>{installation.Size}</span>
                    </div>
                    <div>
                        <span style={style.col1}>Service Env:</span>
                        <span>{installation.ServiceEnvironment}</span>
                    </div>
                    <div>
                        <span style={style.col1}>Created:</span>
                        <span>{installation.CreateAtDate}</span>
                    </div>
                </div>
                {this.installationButtons(installation, shared)}
            </li>
        ));
    }

    render() {
        if (this.props.serverError) {
            return (
                <div style={style.sidebarMessage}>
                    <i className='fa fa-cloud fa-5x'/>
                    <p>{'Received a server error'}</p>
                    <p>{this.props.serverError}</p>
                </div>
            );
        }

        const installs = this.props.installs;
        const sharedInstalls = this.props.sharedInstalls;
        let content;

        if (this.state.serverTypeValue === 'Personal') {
            if (installs.length === 0) {
                content = (
                    <div style={style.sidebarMessage}>
                        <i className='fa fa-cloud fa-5x'/>
                        <p>{'There are no installations. Use the `/cloud create` command to add an installation.'}</p>
                    </div>
                );
            } else {
                content = (
                    <ul style={style.ul}>
                        {this.personalInstallationList(installs)}
                    </ul>
                );
            }
        } else if (sharedInstalls.length === 0) {
            content = (
                <div style={style.sidebarMessage}>
                    <i className='fa fa-cloud fa-5x'/>
                    <p>{'There are no shared installations. Use the `/cloud share` command to share one of yours with other plugin users.'}</p>
                </div>
            );
        } else {
            content = (
                <ul style={style.ul}>
                    {this.sharedInstallationList(sharedInstalls)}
                </ul>
            );
        }

        const serverType = [
            {name: 'Personal', value: 'Personal', icon: 'fa fa-user', count: installs.length},
            {name: 'Shared', value: 'Shared', icon: 'fa fa-users', count: sharedInstalls.length},
        ];

        return (
            <React.Fragment>
                <>
                    {
                        this.state.confirmationModal.visible &&
                            <ConfirmationModal
                                title={this.state.confirmationModal.title}
                                bodyText={this.state.confirmationModal.bodyText}
                                visible={this.state.confirmationModal.visible}
                                onConfirm={this.state.confirmationModal.onConfirm}
                                onCancel={this.state.confirmationModal.onCancel}
                            />
                    }
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
                            <div style={style.serverTypeSelect}>
                                {serverType.map((type, idx) => (
                                    <Button
                                        key={idx}
                                        className={this.state.serverTypeValue === type.value ? 'btn btn-tertiary' : 'btn btn-quaternary'}
                                        value={type.value}
                                        onClick={(e) => this.setServerTypeValue(e.currentTarget.value)}
                                    >
                                        <i className={type.icon}/>
                                        {type.name}
                                        <span
                                            style={style.serverCountBadge}
                                            className='badge'
                                        >{type.count}
                                        </span>
                                    </Button>
                                ))}
                            </div>
                            {content}
                        </div>
                    </Scrollbars>
                </>
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
        marginBottom: '5px',
        fontSize: '16px',
    },
    stable: {
        fontSize: '12px',
        color: 'var(--center-channel-bg)',
        backgroundColor: 'var(--online-indicator)',
        marginRight: '10px',
    },
    inProgress: {
        fontSize: '12px',
        color: 'var(--center-channel-bg)',
        backgroundColor: 'var(--away-indicator)',
        marginRight: '10px',
    },
    sidebarMessage: {
        margin: 'auto',
        width: '50%',
        marginTop: '50px',
        textAlign: 'center',
    },
    dropdownButton: {
        margin: '0 8px',
    },
    serverTypeSelect: {
        padding: '15px',
        borderBottom: '1px solid rgba(var(--center-channel-color-rgb), 0.1)',
    },
    serverCountBadge: {
        color: 'var(--mention-color)',
        backgroundColor: 'var(--mention-bg)',
    },
    badgeIcon: {
        marginLeft: '4px',
    },
};


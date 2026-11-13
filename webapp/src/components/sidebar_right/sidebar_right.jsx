import React from 'react';
import PropTypes from 'prop-types';
import {Button, Scrollbar, Tag} from '@mattermost/compass-ui';

import ConfirmationModal from './confirmation_modal';
import DebugMenu from './debug_menu';

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
        const debugMenuItems = [
            {key: 'installation-logs', onClick: () => window.open(installation.InstallationLogsURL, '_blank'), label: 'Installation Logs'},
            {key: 'provisioner-logs', onClick: () => window.open(installation.ProvisionerLogsURL, '_blank'), label: 'Provisioner Logs'},
            {key: 'debug-packet', onClick: () => this.handleGetDebugPacket(installation.Name), label: 'Get Debug Packet'},
        ];
        let actionButtons;
        if (!shared) {
            actionButtons = [
                <Button
                    data-testid={'restart-' + installation.ID}
                    key={'restart-' + installation.ID}
                    emphasis='tertiary'
                    size='small'
                    onClick={() => this.handleRestartButtonClick(installation)}
                >{'Restart'}
                </Button>,
                this.deletionLockButton(installation),
            ];
        }

        return (
            <div style={style.actions}>
                <Button
                    emphasis='primary'
                    size='small'
                    disabled={!installation.DNSRecords?.length}
                    onClick={() => window.open('https://' + installation.DNSRecords[0].DomainName, '_blank')}
                >{'Open'}
                </Button>
                <DebugMenu
                    title='Debug'
                    items={debugMenuItems}
                />
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
                    key={'unlock-' + installation.ID}
                    emphasis='primary'
                    destructive={true}
                    size='small'
                    onClick={() => this.handleDeletionUnlockButtonClick(installation)}
                >{'Unlock Deletion'}
                </Button>
            );
        }

        return (
            <Button
                key={'lock-' + installation.ID}
                emphasis='tertiary'
                size='small'
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
                    <Tag
                        type={installation.State === 'stable' ? 'success' : 'warning'}
                        casing='title-case'
                        label={(
                            <>
                                {installation.State}
                                <span
                                    style={style.badgeIcon}
                                    className={installation.State === 'stable' ? 'fa fa-check' : 'fa fa-spinner'}
                                />
                            </>
                        )}
                    />
                    {installation.DeletionLocked &&
                    <Tag
                        type='info'
                        label={(
                            <>
                                {'deletion locked'}
                                <span
                                    style={style.badgeIcon}
                                    className='fa fa-lock'
                                />
                            </>
                        )}
                    />
                    }
                    {installation.Shared &&
                    <Tag
                        type='info'
                        label={(
                            <>
                                {installation.AllowSharedUpdates ? 'shared-updates' : 'shared'}
                                <span
                                    style={style.badgeIcon}
                                    className='fa fa-users'
                                />
                            </>
                        )}
                    />
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
                    {installation.ScheduledDeletionTime > 0 && !installation.DeletionLocked && (
                        <div>
                            <span style={style.col1}>Deleting:</span>
                            <span
                                style={this.getTimeRemainingStyle(installation.ScheduledDeletionTime)}
                                title={new Date(installation.ScheduledDeletionTime).toLocaleString()}
                            >
                                {this.formatDeletionTime(installation.ScheduledDeletionTime)}
                                <i
                                    className='fa fa-clock-o'
                                    style={{marginLeft: '4px'}}
                                />
                            </span>
                        </div>
                    )}
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
                    <Scrollbar
                        className='SidebarRight'
                        style={style.scrollbar}
                    >
                        <div style={style.container}>
                            <div style={style.serverTypeSelect}>
                                {serverType.map((type, idx) => (
                                    <Button
                                        key={idx}
                                        type='button'
                                        emphasis={this.state.serverTypeValue === type.value ? 'tertiary' : 'quaternary'}
                                        aria-pressed={this.state.serverTypeValue === type.value}
                                        leadingIcon={<i className={type.icon}/>}
                                        trailingIcon={(
                                            <Tag
                                                type='info-dim'
                                                label={String(type.count)}
                                            />
                                        )}
                                        onClick={() => this.setServerTypeValue(type.value)}
                                    >
                                        {type.name}
                                    </Button>
                                ))}
                            </div>
                            {content}
                        </div>
                    </Scrollbar>
                </>
            </React.Fragment>
        );
    }

    formatDeletionTime(timestamp) {
        const now = new Date();
        const deletionTime = new Date(timestamp);
        const diffInMs = deletionTime - now;
        const diffInMinutes = Math.floor(diffInMs / (1000 * 60));
        const diffInHours = Math.floor(diffInMinutes / 60);
        const diffInDays = Math.floor(diffInHours / 24);
        const remainingHours = diffInHours % 24;

        if (diffInDays > 0) {
            if (remainingHours > 0) {
                return `${diffInDays} day${diffInDays > 1 ? 's' : ''} and ${remainingHours} hour${remainingHours > 1 ? 's' : ''}`;
            }
            return `${diffInDays} day${diffInDays > 1 ? 's' : ''}`;
        } else if (diffInHours > 0) {
            return `${diffInHours} hour${diffInHours > 1 ? 's' : ''}`;
        } else if (diffInMinutes > 0) {
            return `${diffInMinutes} minute${diffInMinutes > 1 ? 's' : ''}`;
        }
        return 'less than a minute';
    }

    getTimeRemainingStyle(timestamp) {
        const now = new Date();
        const deletionTime = new Date(timestamp);
        const diffInMs = deletionTime - now;
        const diffInMinutes = Math.floor(diffInMs / (1000 * 60));
        const diffInHours = Math.floor(diffInMinutes / 60);
        const diffInDays = Math.floor(diffInHours / 24);

        // Apply urgent styling (red) only when less than a day remains
        if (diffInDays > 0) {
            return style.normalDeletionTime;
        }
        return style.urgentDeletionTime;
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
        flexWrap: 'wrap',
        gap: '8px',
        marginBottom: '10px',
    },
    actions: {
        display: 'flex',
        flexWrap: 'wrap',
        gap: '8px',
    },
    scrollbar: {
        height: '100%',
    },
    installinfo: {
        fontSize: '12px',
        marginBottom: '15px',
    },
    nameText: {
        marginBottom: '5px',
        fontSize: '16px',
    },
    sidebarMessage: {
        margin: 'auto',
        width: '50%',
        marginTop: '50px',
        textAlign: 'center',
    },
    serverTypeSelect: {
        display: 'flex',
        gap: '8px',
        padding: '15px',
        borderBottom: '1px solid rgba(var(--center-channel-color-rgb), 0.1)',
    },
    badgeIcon: {
        marginLeft: '4px',
    },
    normalDeletionTime: {
        color: 'var(--center-channel-color)',
        fontWeight: 'normal',
    },
    urgentDeletionTime: {
        color: 'var(--error-text)',
        fontWeight: 'bold',
    },
};


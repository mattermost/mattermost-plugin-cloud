import React from 'react';
import PropTypes from 'prop-types';
import {Button, Icon, SectionNotice, Switch} from '@mattermost/compass-ui';
import OpenInNewIcon from '@mattermost/compass-icons/components/open-in-new';
import PencilOutlineIcon from '@mattermost/compass-icons/components/pencil-outline';

import DebugMenu from './debug_menu';
import InstallationConfirmation from './installation_confirmation';
import InstallationBadges from './installation_badges';
import {
    canUpdateInstallation,
    formatDeletionTime,
    getDomain,
    getVersionLabel,
    isDeletionUrgent,
    isDeleted,
    isOwner,
} from './installation_utils';
import useInstallationActions, {getDebugMenuItems} from './use_installation_actions';

function InfoRow({label, children}) {
    return (
        <>
            <dt className='CloudRHS__infoLabel'>{label}</dt>
            <dd className='CloudRHS__infoValue'>{children}</dd>
        </>
    );
}

InfoRow.propTypes = {
    label: PropTypes.string.isRequired,
    children: PropTypes.node,
};

export default function InstallationDetail({
    installation,
    userID,
    lockedCount,
    maxLockedInstallations,
    actions,
    onEdit,
    onDeleted,
}) {
    const {busyAction, busy, error, clearError, confirmation, setConfirmation, run, confirm} = useInstallationActions({installation, actions, onDeleted});

    const owner = isOwner(installation, userID);
    const canUpdate = canUpdateInstallation(installation, userID);
    const deleted = isDeleted(installation);
    const domain = getDomain(installation);
    const lockLimitReached = !installation.DeletionLocked && lockedCount >= maxLockedInstallations;

    const debugMenuItems = getDebugMenuItems(installation, actions);

    return (
        <div className='CloudRHS__detail'>
            {confirmation && (
                <InstallationConfirmation
                    installation={installation}
                    confirmation={confirmation}
                    onConfirm={confirm}
                    onCancel={() => setConfirmation('')}
                />
            )}

            <div className='CloudRHS__detailHeader'>
                <h2 className='CloudRHS__detailName'>{installation.Name}</h2>
                <InstallationBadges installation={installation}/>
            </div>

            {error && (
                <SectionNotice
                    type='danger'
                    title='The action failed'
                    description={error}
                    onDismiss={clearError}
                />
            )}

            <div className='CloudRHS__actions'>
                <Button
                    size='small'
                    emphasis='primary'
                    disabled={!domain || deleted}
                    leadingIcon={<Icon glyph={<OpenInNewIcon/>}/>}
                    onClick={() => window.open('https://' + domain, '_blank')}
                >
                    {'Open'}
                </Button>
                {canUpdate && (
                    <Button
                        size='small'
                        emphasis='secondary'
                        disabled={busy || deleted}
                        leadingIcon={<Icon glyph={<PencilOutlineIcon/>}/>}
                        onClick={onEdit}
                    >
                        {'Edit'}
                    </Button>
                )}
                <DebugMenu
                    title='Debug'
                    items={debugMenuItems}
                />
            </div>

            <dl className='CloudRHS__info'>
                <InfoRow label='DNS'>{domain || 'No URL!'}</InfoRow>
                <InfoRow label='Image'>{installation.Image}</InfoRow>
                <InfoRow label={installation.Tag ? 'Tag' : 'Version'}>{getVersionLabel(installation)}</InfoRow>
                <InfoRow label='Database'>{installation.Database}</InfoRow>
                <InfoRow label='Filestore'>{installation.Filestore}</InfoRow>
                <InfoRow label='Size'>{installation.Size}</InfoRow>
                <InfoRow label='Affinity'>{installation.Affinity}</InfoRow>
                <InfoRow label='Service Env'>{installation.ServiceEnvironment}</InfoRow>
                <InfoRow label='Created'>{installation.CreateAtDate}</InfoRow>
                {installation.ScheduledDeletionTime > 0 && !installation.DeletionLocked && (
                    <InfoRow label='Deleting'>
                        <span
                            className={isDeletionUrgent(installation.ScheduledDeletionTime) ? 'CloudRHS__deletion CloudRHS__deletion--urgent' : 'CloudRHS__deletion'}
                            title={new Date(installation.ScheduledDeletionTime).toLocaleString()}
                        >
                            {formatDeletionTime(installation.ScheduledDeletionTime)}
                        </span>
                    </InfoRow>
                )}
            </dl>

            {canUpdate && !deleted && (
                <section className='CloudRHS__section'>
                    <h3 className='CloudRHS__sectionTitle'>{'Manage'}</h3>
                    <div className='CloudRHS__actions'>
                        <Button
                            size='small'
                            emphasis='tertiary'
                            loading={busyAction === 'restart'}
                            disabled={busy}
                            onClick={() => setConfirmation('restart')}
                        >
                            {'Restart'}
                        </Button>
                        {owner && installation.State === 'stable' && (
                            <Button
                                size='small'
                                emphasis='tertiary'
                                loading={busyAction === 'hibernate'}
                                disabled={busy}
                                onClick={() => setConfirmation('hibernate')}
                            >
                                {'Hibernate'}
                            </Button>
                        )}
                        {owner && installation.State === 'hibernating' && (
                            <Button
                                size='small'
                                emphasis='tertiary'
                                loading={busyAction === 'wake'}
                                disabled={busy}
                                onClick={() => run('wake', () => actions.wakeInstallation(installation.ID))}
                            >
                                {'Wake up'}
                            </Button>
                        )}
                        {owner && installation.DeletionLocked && (
                            <Button
                                size='small'
                                emphasis='primary'
                                destructive={true}
                                loading={busyAction === 'unlock'}
                                disabled={busy}
                                onClick={() => setConfirmation('unlock')}
                            >
                                {'Unlock Deletion'}
                            </Button>
                        )}
                        {owner && !installation.DeletionLocked && (
                            <Button
                                size='small'
                                emphasis='tertiary'
                                loading={busyAction === 'lock'}
                                disabled={busy || lockLimitReached}
                                onClick={() => run('lock', () => actions.deletionLockInstallation(installation.ID))}
                            >
                                {'Lock Deletion'}
                            </Button>
                        )}
                    </div>
                </section>
            )}

            {owner && !deleted && (
                <section className='CloudRHS__section'>
                    <h3 className='CloudRHS__sectionTitle'>{'Sharing'}</h3>
                    <Switch
                        checked={Boolean(installation.Shared)}
                        disabled={busy}
                        secondaryLabel='Other plugin users can see this server.'
                        onChange={(e) => run('share', () => actions.shareInstallation(installation.ID, e.target.checked, false))}
                    >
                        {'Share with other plugin users'}
                    </Switch>
                    <Switch
                        checked={Boolean(installation.AllowSharedUpdates)}
                        disabled={busy || !installation.Shared}
                        secondaryLabel='Other plugin users can update and restart this server.'
                        onChange={(e) => run('share', () => actions.shareInstallation(installation.ID, true, e.target.checked))}
                    >
                        {'Allow shared updates'}
                    </Switch>
                </section>
            )}

            {owner && !deleted && (
                <section className='CloudRHS__section'>
                    <h3 className='CloudRHS__sectionTitle'>{'Danger zone'}</h3>
                    <div className='CloudRHS__actions'>
                        <Button
                            size='small'
                            emphasis='primary'
                            destructive={true}
                            loading={busyAction === 'delete'}
                            disabled={busy || installation.DeletionLocked}
                            onClick={() => setConfirmation('delete')}
                        >
                            {'Delete'}
                        </Button>
                    </div>
                    {installation.DeletionLocked && (
                        <p className='CloudRHS__hint'>{'Unlock deletion to delete this server.'}</p>
                    )}
                </section>
            )}
        </div>
    );
}

InstallationDetail.propTypes = {
    installation: PropTypes.object.isRequired,
    userID: PropTypes.string.isRequired,
    lockedCount: PropTypes.number.isRequired,
    maxLockedInstallations: PropTypes.number,
    actions: PropTypes.shape({
        restartInstallation: PropTypes.func.isRequired,
        hibernateInstallation: PropTypes.func.isRequired,
        wakeInstallation: PropTypes.func.isRequired,
        deleteInstallation: PropTypes.func.isRequired,
        shareInstallation: PropTypes.func.isRequired,
        deletionLockInstallation: PropTypes.func.isRequired,
        deletionUnlockInstallation: PropTypes.func.isRequired,
        getDebugPacket: PropTypes.func.isRequired,
    }).isRequired,
    onEdit: PropTypes.func.isRequired,
    onDeleted: PropTypes.func.isRequired,
};

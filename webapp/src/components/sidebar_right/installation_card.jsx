import React from 'react';
import PropTypes from 'prop-types';
import {Button, Icon, IconButton, SectionNotice} from '@mattermost/compass-ui';
import ChevronRightIcon from '@mattermost/compass-icons/components/chevron-right';
import OpenInNewIcon from '@mattermost/compass-icons/components/open-in-new';
import PlusIcon from '@mattermost/compass-icons/components/plus';

import DebugMenu from './debug_menu';
import InstallationBadges from './installation_badges';
import InstallationConfirmation from './installation_confirmation';
import {canUpdateInstallation, getDomain, getVersionLabel, isDeleted, isOwner} from './installation_utils';
import useInstallationActions, {getDebugMenuItems} from './use_installation_actions';

// The name button covers the whole card, so the domain link can sit above it without nesting a link in a button.
export function InstallationCard({installation, userID, lockedCount, maxLockedInstallations, actions, onSelect}) {
    const domain = getDomain(installation);
    const version = getVersionLabel(installation);
    const {busyAction, busy, error, clearError, confirmation, setConfirmation, run, confirm} = useInstallationActions({installation, actions});

    const deleted = isDeleted(installation);
    const owner = isOwner(installation, userID);
    const canUpdate = canUpdateInstallation(installation, userID);
    const lockLimitReached = !installation.DeletionLocked && lockedCount >= maxLockedInstallations;

    return (
        <div
            className='CloudRHS__card'
            data-testid={'installation-card-' + installation.ID}
        >
            {confirmation && (
                <InstallationConfirmation
                    installation={installation}
                    confirmation={confirmation}
                    onConfirm={confirm}
                    onCancel={() => setConfirmation('')}
                />
            )}
            <div className='CloudRHS__cardBody'>
                <div className='CloudRHS__cardTitle'>
                    <button
                        type='button'
                        className='CloudRHS__cardSelect'
                        onClick={() => onSelect(installation)}
                    >
                        {installation.Name}
                    </button>
                    {version && <span className='CloudRHS__cardVersion'>{version}</span>}
                </div>
                <InstallationBadges installation={installation}/>
                <span className='CloudRHS__cardDomain'>
                    {domain && (
                        <IconButton
                            asChild={true}
                            size='x-small'
                            aria-label={`Open ${domain} in a new tab`}
                            icon={<Icon glyph={<OpenInNewIcon/>}/>}
                        >
                            <a
                                href={'https://' + domain}
                                target='_blank'
                                rel='noopener noreferrer'
                            />
                        </IconButton>
                    )}
                    <span className='CloudRHS__cardMeta'>{domain || 'No URL!'}</span>
                </span>
                {!deleted && (
                    <div className='CloudRHS__cardActions'>
                        <DebugMenu
                            title='Debug'
                            items={getDebugMenuItems(installation, actions)}
                        />
                        {canUpdate && (
                            <Button
                                size='small'
                                emphasis='tertiary'
                                loading={busyAction === 'restart'}
                                disabled={busy}
                                onClick={() => setConfirmation('restart')}
                            >
                                {'Restart'}
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
                )}
                {error && (
                    <div className='CloudRHS__cardError'>
                        <SectionNotice
                            type='danger'
                            title='The action failed'
                            description={error}
                            onDismiss={clearError}
                        />
                    </div>
                )}
            </div>
            <span className='CloudRHS__cardChevron'>
                <Icon
                    glyph={<ChevronRightIcon/>}
                    size='20'
                />
            </span>
        </div>
    );
}

InstallationCard.propTypes = {
    installation: PropTypes.object.isRequired,
    userID: PropTypes.string.isRequired,
    lockedCount: PropTypes.number.isRequired,
    maxLockedInstallations: PropTypes.number,
    actions: PropTypes.object.isRequired,
    onSelect: PropTypes.func.isRequired,
};

export function AddInstallationCard({onClick}) {
    return (
        <button
            type='button'
            className='CloudRHS__card CloudRHS__card--add'
            aria-label='Add new server'
            data-testid='add-installation'
            onClick={onClick}
        >
            <Icon
                glyph={<PlusIcon/>}
                size='64'
            />
        </button>
    );
}

AddInstallationCard.propTypes = {
    onClick: PropTypes.func.isRequired,
};

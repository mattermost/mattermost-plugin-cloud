import React from 'react';
import PropTypes from 'prop-types';
import {Icon, Spinner, Tag} from '@mattermost/compass-ui';
import AccountMultipleOutlineIcon from '@mattermost/compass-icons/components/account-multiple-outline';
import AlertCircleOutlineIcon from '@mattermost/compass-icons/components/alert-circle-outline';
import CheckCircleOutlineIcon from '@mattermost/compass-icons/components/check-circle-outline';
import LockOutlineIcon from '@mattermost/compass-icons/components/lock-outline';
import PauseIcon from '@mattermost/compass-icons/components/pause';

import {isFailedState, isTransitionalState, stateLabel, stateTone} from './installation_utils';

function stateIcon(state) {
    if (state === 'stable') {
        return <Icon glyph={<CheckCircleOutlineIcon/>}/>;
    }
    if (state === 'hibernating') {
        return <Icon glyph={<PauseIcon/>}/>;
    }
    if (isFailedState(state)) {
        return <Icon glyph={<AlertCircleOutlineIcon/>}/>;
    }
    if (isTransitionalState(state)) {
        return <Spinner size='10'/>;
    }
    return null;
}

export function StateBadge({state}) {
    return (
        <Tag
            data-testid='state-badge'
            type={stateTone(state)}
            size='small'
            casing='title-case'
            label={stateLabel(state)}
            leadingIcon={stateIcon(state)}
        />
    );
}

StateBadge.propTypes = {
    state: PropTypes.string,
};

export default function InstallationBadges({installation}) {
    return (
        <div className='CloudRHS__badges'>
            <StateBadge state={installation.State}/>
            {installation.DeletionLocked && (
                <Tag
                    type='info'
                    size='small'
                    casing='title-case'
                    label={'deletion locked'}
                    leadingIcon={<Icon glyph={<LockOutlineIcon/>}/>}
                />
            )}
            {installation.Shared && (
                <Tag
                    type='info'
                    size='small'
                    casing='title-case'
                    label={installation.AllowSharedUpdates ? 'shared-updates' : 'shared'}
                    leadingIcon={<Icon glyph={<AccountMultipleOutlineIcon/>}/>}
                />
            )}
        </div>
    );
}

InstallationBadges.propTypes = {
    installation: PropTypes.object.isRequired,
};

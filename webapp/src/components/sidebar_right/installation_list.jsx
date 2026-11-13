import React from 'react';
import PropTypes from 'prop-types';
import {EmptyState, SearchInput, Tabs} from '@mattermost/compass-ui';

import {AddInstallationCard, InstallationCard} from './installation_card';
import {matchesQuery} from './installation_utils';

export const TAB_MINE = 'mine';
export const TAB_SHARED = 'shared';

export default function InstallationList({userID, lockedCount, maxLockedInstallations, actions, installs, sharedInstalls, tab, onTabChange, query, onQueryChange, onSelect, onAdd}) {
    const tabs = [
        {key: TAB_MINE, label: 'Mine', countBadge: installs.length, buttonProps: {'data-testid': 'tab-mine'}},
        {key: TAB_SHARED, label: 'Shared', countBadge: sharedInstalls.length, buttonProps: {'data-testid': 'tab-shared'}},
    ];

    const mine = tab === TAB_MINE;
    const source = mine ? installs : sharedInstalls;
    const visible = source.filter((installation) => matchesQuery(installation, query));
    const noMatches = source.length > 0 && visible.length === 0;

    return (
        <div>
            <Tabs
                className='CloudRHS__tabs'
                tabs={tabs}
                activeKey={tab}
                appearance='underlined'
                onChange={onTabChange}
            />
            {source.length > 0 && (
                <div className='CloudRHS__search'>
                    <SearchInput
                        label='Search servers'
                        aria-label='Search servers'
                        value={query}
                        onChange={(e) => onQueryChange(e.target.value)}
                        onClear={() => onQueryChange('')}
                    />
                </div>
            )}
            {noMatches && (
                <EmptyState
                    className='CloudRHS__empty'
                    title='No matching servers'
                    description='Try a different name, domain, or version.'
                />
            )}
            {source.length === 0 && (
                <EmptyState
                    className='CloudRHS__empty'
                    title={mine ? 'No servers yet' : 'No shared servers'}
                    description={mine ? 'Add a server to get started.' : 'Servers that other plugin users share with you appear here.'}
                />
            )}
            <ul className='CloudRHS__list'>
                {visible.map((installation) => (
                    <li key={installation.ID}>
                        <InstallationCard
                            installation={installation}
                            userID={userID}
                            lockedCount={lockedCount}
                            maxLockedInstallations={maxLockedInstallations}
                            actions={actions}
                            onSelect={onSelect}
                        />
                    </li>
                ))}
                {mine && (
                    <li>
                        <AddInstallationCard onClick={onAdd}/>
                    </li>
                )}
            </ul>
        </div>
    );
}

InstallationList.propTypes = {
    userID: PropTypes.string.isRequired,
    lockedCount: PropTypes.number.isRequired,
    maxLockedInstallations: PropTypes.number,
    actions: PropTypes.object.isRequired,
    installs: PropTypes.array.isRequired,
    sharedInstalls: PropTypes.array.isRequired,
    tab: PropTypes.oneOf([TAB_MINE, TAB_SHARED]).isRequired,
    onTabChange: PropTypes.func.isRequired,
    query: PropTypes.string.isRequired,
    onQueryChange: PropTypes.func.isRequired,
    onSelect: PropTypes.func.isRequired,
    onAdd: PropTypes.func.isRequired,
};

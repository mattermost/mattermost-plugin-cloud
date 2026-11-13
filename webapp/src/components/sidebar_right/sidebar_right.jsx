import React, {useCallback, useEffect, useState} from 'react';
import PropTypes from 'prop-types';
import {Button, EmptyState, Icon, Scrollbar, Spinner} from '@mattermost/compass-ui';
import ArrowLeftIcon from '@mattermost/compass-icons/components/arrow-left';

import InstallationDetail from './installation_detail';
import InstallationForm from './installation_form';
import InstallationList, {TAB_MINE} from './installation_list';
import {isTransitionalState} from './installation_utils';
import './sidebar_right.scss';

const VIEW_LIST = 'list';
const VIEW_DETAIL = 'detail';
const VIEW_EDIT = 'edit';
const VIEW_CREATE = 'create';

const LIST_ROUTE = {view: VIEW_LIST};

// Fallback for when provisioner webhooks do not reach the plugin, for example when none is registered.
export const POLL_INTERVAL_MILLISECONDS = 15000;

export default function SidebarRight({id, installs, sharedInstalls, sharedError, serverError, maxLockedInstallations, options, actions}) {
    const [stack, setStack] = useState([LIST_ROUTE]);
    const [tab, setTab] = useState(TAB_MINE);
    const [query, setQuery] = useState('');
    const [loading, setLoading] = useState(true);

    const {setVisible, getCloudUserData, getSharedInstalls, getPluginConfiguration} = actions;

    useEffect(() => {
        setVisible(true);
        Promise.all([
            getCloudUserData(id),
            getSharedInstalls(),
            getPluginConfiguration(),
        ]).finally(() => setLoading(false));

        return () => setVisible(false);
    }, [id, setVisible, getCloudUserData, getSharedInstalls, getPluginConfiguration]);

    const pollMine = installs.some((install) => isTransitionalState(install.State));
    const pollShared = sharedInstalls.some((install) => isTransitionalState(install.State));
    const {refreshInstallations} = actions;

    useEffect(() => {
        let timer = null;
        if (pollMine || pollShared) {
            timer = setInterval(() => refreshInstallations({mine: pollMine, shared: pollShared}), POLL_INTERVAL_MILLISECONDS);
        }

        return () => clearInterval(timer);
    }, [pollMine, pollShared, refreshInstallations]);

    const route = stack[stack.length - 1];
    const push = useCallback((next) => setStack((current) => [...current, next]), []);
    const back = useCallback(() => setStack((current) => (current.length > 1 ? current.slice(0, -1) : current)), []);
    const home = useCallback(() => setStack([LIST_ROUTE]), []);

    if (serverError) {
        return (
            <EmptyState
                className='CloudRHS__empty'
                title='Received a server error'
                description={serverError}
            />
        );
    }

    if (loading) {
        return (
            <div className='CloudRHS__loading'>
                <Spinner size='32'/>
            </div>
        );
    }

    const lockedCount = installs.filter((install) => install.DeletionLocked).length;
    const all = [...installs, ...sharedInstalls];
    const installation = route.installationID ? all.find((item) => item.ID === route.installationID) : null;

    // A selected installation can disappear after a refresh, for example when someone unshares it.
    if (route.installationID && !installation) {
        return (
            <EmptyState
                className='CloudRHS__empty'
                title='Server not found'
                description='This server no longer exists or is no longer shared with you.'
                action={{children: 'Back to list', onClick: home}}
            />
        );
    }

    const onCreate = async (payload) => {
        const result = await actions.createInstallation(payload);
        if (!result.error) {
            setTab(TAB_MINE);
            home();
        }
        return result;
    };

    const onUpdate = async (payload) => {
        const result = await actions.updateInstallation(payload);
        if (!result.error) {
            back();
        }
        return result;
    };

    let title = '';
    let content;
    switch (route.view) {
    case VIEW_DETAIL:
        title = installation.Name;
        content = (
            <InstallationDetail
                key={installation.ID}
                installation={installation}
                userID={id}
                lockedCount={lockedCount}
                maxLockedInstallations={maxLockedInstallations}
                actions={actions}
                onEdit={() => push({view: VIEW_EDIT, installationID: installation.ID})}
                onDeleted={home}
            />
        );
        break;
    case VIEW_EDIT:
        title = 'Edit ' + installation.Name;
        content = (
            <InstallationForm
                mode='edit'
                installation={installation}
                options={options}
                onSubmit={onUpdate}
                onCancel={back}
            />
        );
        break;
    case VIEW_CREATE:
        title = 'Add server';
        content = (
            <InstallationForm
                mode='create'
                options={options}
                onSubmit={onCreate}
                onCancel={back}
            />
        );
        break;
    default:
        content = (
            <InstallationList
                userID={id}
                lockedCount={lockedCount}
                maxLockedInstallations={maxLockedInstallations}
                actions={actions}
                installs={installs}
                sharedInstalls={sharedInstalls}
                sharedError={sharedError}
                onRetryShared={actions.getSharedInstalls}
                tab={tab}
                onTabChange={setTab}
                query={query}
                onQueryChange={setQuery}
                onSelect={(item) => push({view: VIEW_DETAIL, installationID: item.ID})}
                onAdd={() => push({view: VIEW_CREATE})}
            />
        );
    }

    return (
        <Scrollbar
            className='SidebarRight'
            style={{height: '100%'}}
        >
            {route.view !== VIEW_LIST && (
                <div className='CloudRHS__header'>
                    <Button
                        type='button'
                        emphasis='tertiary'
                        size='small'
                        aria-label='Back'
                        leadingIcon={<Icon glyph={<ArrowLeftIcon/>}/>}
                        onClick={back}
                    >
                        {'Back'}
                    </Button>
                    <span className='CloudRHS__title'>{title}</span>
                </div>
            )}
            {content}
        </Scrollbar>
    );
}

SidebarRight.propTypes = {
    id: PropTypes.string.isRequired,
    installs: PropTypes.array.isRequired,
    sharedInstalls: PropTypes.array.isRequired,
    sharedError: PropTypes.string,
    serverError: PropTypes.string.isRequired,
    maxLockedInstallations: PropTypes.number,
    options: PropTypes.object,
    actions: PropTypes.shape({
        setVisible: PropTypes.func.isRequired,
        getCloudUserData: PropTypes.func.isRequired,
        getSharedInstalls: PropTypes.func.isRequired,
        getPluginConfiguration: PropTypes.func.isRequired,
        refreshInstallations: PropTypes.func.isRequired,
        createInstallation: PropTypes.func.isRequired,
        updateInstallation: PropTypes.func.isRequired,
        restartInstallation: PropTypes.func.isRequired,
        hibernateInstallation: PropTypes.func.isRequired,
        wakeInstallation: PropTypes.func.isRequired,
        deleteInstallation: PropTypes.func.isRequired,
        shareInstallation: PropTypes.func.isRequired,
        deletionLockInstallation: PropTypes.func.isRequired,
        deletionUnlockInstallation: PropTypes.func.isRequired,
        getDebugPacket: PropTypes.func.isRequired,
    }).isRequired,
};

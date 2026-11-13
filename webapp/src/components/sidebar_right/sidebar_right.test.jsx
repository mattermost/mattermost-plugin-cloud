import React from 'react';
import {act, render, screen, fireEvent, waitFor, within} from '@testing-library/react';

import SidebarRight, {POLL_INTERVAL_MILLISECONDS} from './sidebar_right';

const baseInstall = {
    Image: 'mattermost/mattermost-enterprise-edition',
    Tag: '9.5.0',
    Version: 'sha256:abc',
    Database: 'aws-multitenant-rds-postgres-pgbouncer',
    Filestore: 'bifrost',
    Size: 'miniSingleton',
    Affinity: 'multitenant',
    InstallationLogsURL: 'https://logs.example.com/installation',
    ProvisionerLogsURL: 'https://logs.example.com/provisioner',
    DeletionLocked: false,
};

const options = {
    Sizes: ['miniSingleton', 'miniHA'],
    Licenses: ['enterprise', 'professional'],
    Images: ['mattermost/mattermost-enterprise-edition'],
    Affinities: ['multitenant', 'isolated'],
    Databases: ['aws-multitenant-rds-postgres-pgbouncer'],
    Filestores: ['bifrost'],
    DefaultSize: 'miniSingleton',
    DefaultVersion: 'latest',
    DefaultLicense: 'enterprise',
    DefaultImage: 'mattermost/mattermost-enterprise-edition',
    DefaultAffinity: 'multitenant',
    DefaultDatabase: 'aws-multitenant-rds-postgres-pgbouncer',
    DefaultFilestore: 'bifrost',
};

function makeProps(overrides = {}) {
    const ok = () => jest.fn().mockResolvedValue({data: {}});
    return {
        id: 'test-id',
        installs: [
            {...baseInstall, ID: '1', Name: 'first', OwnerID: 'test-id', State: 'stable', DNSRecords: [{DomainName: 'first.example.com'}]},
            {...baseInstall, ID: '2', Name: 'second', OwnerID: 'test-id', State: 'update-in-progress', DeletionLocked: true, DNSRecords: [{DomainName: 'second.example.com'}]},
            {...baseInstall, ID: '4', Name: 'sleeping', OwnerID: 'test-id', State: 'hibernating', DNSRecords: []},
        ],
        sharedInstalls: [
            {...baseInstall, ID: '3', Name: 'borrowed', OwnerID: 'other', State: 'stable', Shared: true, AllowSharedUpdates: true, DNSRecords: [{DomainName: 'borrowed.example.com'}]},
            {...baseInstall, ID: '5', Name: 'readonly', OwnerID: 'other', State: 'stable', Shared: true, AllowSharedUpdates: false, DNSRecords: []},
        ],
        serverError: '',
        maxLockedInstallations: 2,
        options,
        actions: {
            setVisible: jest.fn(),
            getCloudUserData: jest.fn().mockResolvedValue({}),
            getSharedInstalls: jest.fn().mockResolvedValue({}),
            getPluginConfiguration: jest.fn().mockResolvedValue({}),
            refreshInstallations: jest.fn().mockResolvedValue({}),
            getDebugPacket: jest.fn(),
            createInstallation: ok(),
            updateInstallation: ok(),
            restartInstallation: ok(),
            hibernateInstallation: ok(),
            wakeInstallation: ok(),
            deleteInstallation: ok(),
            shareInstallation: ok(),
            deletionLockInstallation: ok(),
            deletionUnlockInstallation: ok(),
        },
        ...overrides,
    };
}

async function renderLoaded(props) {
    let view;
    await act(async () => {
        view = render(<SidebarRight {...props}/>);
    });
    return view;
}

function openInstallation(name) {
    fireEvent.click(screen.getByText(name));
}

describe('SidebarRight list', () => {
    it('loads data and marks the RHS visible on mount', async () => {
        const props = makeProps();
        await renderLoaded(props);

        expect(props.actions.setVisible).toHaveBeenCalledWith(true);
        expect(props.actions.getCloudUserData).toHaveBeenCalledWith('test-id');
        expect(props.actions.getSharedInstalls).toHaveBeenCalled();
        expect(props.actions.getPluginConfiguration).toHaveBeenCalled();
    });

    it('marks the RHS hidden on unmount', async () => {
        const props = makeProps();
        const {unmount} = await renderLoaded(props);

        unmount();

        expect(props.actions.setVisible).toHaveBeenLastCalledWith(false);
    });

    it('shows a spinner until the first load finishes', async () => {
        const props = makeProps();
        let resolveLoad;
        props.actions.getSharedInstalls = jest.fn(() => new Promise((resolve) => {
            resolveLoad = resolve;
        }));
        render(<SidebarRight {...props}/>);

        expect(screen.getByRole('status')).toBeInTheDocument();
        expect(screen.queryByTestId('tab-mine')).not.toBeInTheDocument();

        await act(async () => resolveLoad({}));
        expect(screen.getByTestId('tab-mine')).toBeInTheDocument();
    });

    it('lists own installations with state badges and an add card', async () => {
        await renderLoaded(makeProps());

        expect(screen.getByText('first')).toBeInTheDocument();
        expect(screen.getByText('second')).toBeInTheDocument();
        expect(screen.queryByText('borrowed')).not.toBeInTheDocument();
        expect(screen.getByTestId('add-installation')).toBeInTheDocument();

        const badges = screen.getAllByTestId('state-badge').map((badge) => badge.textContent);
        expect(badges).toEqual(['stable', 'update in progress', 'hibernating']);
    });

    it('lists shared installations without an add card', async () => {
        await renderLoaded(makeProps());

        fireEvent.click(screen.getByTestId('tab-shared'));

        expect(screen.getByText('borrowed')).toBeInTheDocument();
        expect(screen.queryByText('first')).not.toBeInTheDocument();
        expect(screen.queryByTestId('add-installation')).not.toBeInTheDocument();
    });

    it('explains an empty shared list', async () => {
        await renderLoaded(makeProps({sharedInstalls: []}));

        fireEvent.click(screen.getByTestId('tab-shared'));

        expect(screen.getByText('No shared servers')).toBeInTheDocument();
    });

    it('still offers the add card when the user has no installations', async () => {
        await renderLoaded(makeProps({installs: []}));

        expect(screen.getByText('No servers yet')).toBeInTheDocument();
        expect(screen.getByTestId('add-installation')).toBeInTheDocument();
    });

    it('shows a server error instead of the list', async () => {
        await renderLoaded(makeProps({serverError: 'Status: 500, Message: boom'}));

        expect(screen.getByText('Received a server error')).toBeInTheDocument();
        expect(screen.getByText('Status: 500, Message: boom')).toBeInTheDocument();
    });

    it('reflects a state change from props without a reload', async () => {
        const props = makeProps();
        const {rerender} = await renderLoaded(props);
        expect(screen.getAllByTestId('state-badge')[1]).toHaveTextContent('update in progress');

        const installs = props.installs.map((install) => (install.ID === '2' ? {...install, State: 'stable'} : install));
        rerender(
            <SidebarRight
                {...props}
                installs={installs}
            />,
        );

        expect(screen.getAllByTestId('state-badge')[1]).toHaveTextContent('stable');
    });
});

describe('SidebarRight polling fallback', () => {
    beforeEach(() => {
        jest.useFakeTimers();
    });

    afterEach(() => {
        jest.useRealTimers();
    });

    const tick = async () => {
        await act(async () => {
            jest.advanceTimersByTime(POLL_INTERVAL_MILLISECONDS);
        });
    };

    it('reloads own installations while one is changing state', async () => {
        const props = makeProps();
        await renderLoaded(props);

        await tick();

        expect(props.actions.refreshInstallations).toHaveBeenCalledWith({mine: true, shared: false});
    });

    it('stops reloading when every installation is settled', async () => {
        const props = makeProps();
        const {rerender} = await renderLoaded(props);
        await tick();
        props.actions.refreshInstallations.mockClear();

        const settled = props.installs.map((install) => ({...install, State: 'stable'}));
        rerender(
            <SidebarRight
                {...props}
                installs={settled}
            />,
        );
        await tick();

        expect(props.actions.refreshInstallations).not.toHaveBeenCalled();
    });

    it('reloads shared installations while one is changing state', async () => {
        const props = makeProps();
        props.installs = props.installs.map((install) => ({...install, State: 'stable'}));
        props.sharedInstalls = props.sharedInstalls.map((install) => ({...install, State: 'update-in-progress'}));
        await renderLoaded(props);

        await tick();

        expect(props.actions.refreshInstallations).toHaveBeenCalledWith({mine: false, shared: true});
    });

    it('does not reload when nothing is changing', async () => {
        const props = makeProps();
        props.installs = props.installs.map((install) => ({...install, State: 'stable'}));
        await renderLoaded(props);

        await tick();

        expect(props.actions.refreshInstallations).not.toHaveBeenCalled();
    });

    it('does not poll for a server that waits for deletion', async () => {
        const props = makeProps();
        props.installs = props.installs.map((install) => ({...install, State: 'deletion-pending'}));
        await renderLoaded(props);

        await tick();

        expect(props.actions.refreshInstallations).not.toHaveBeenCalled();
    });
});

describe('SidebarRight card content', () => {
    it('shows the version next to the name and no size', async () => {
        await renderLoaded(makeProps());

        const card = screen.getByTestId('installation-card-1');
        const title = within(card).getByText('first').parentElement;

        expect(title).toHaveTextContent('first9.5.0');
        expect(within(card).queryByText(/miniSingleton/)).not.toBeInTheDocument();
    });

    it('falls back to the version digest when there is no tag', async () => {
        const props = makeProps();
        props.installs = [{...props.installs[0], Tag: ''}];
        await renderLoaded(props);

        expect(within(screen.getByTestId('installation-card-1')).getByText('sha256:abc')).toBeInTheDocument();
    });
});

describe('SidebarRight card actions', () => {
    const card = (id) => within(screen.getByTestId('installation-card-' + id));

    it('offers Debug, Restart, and lock actions on own cards', async () => {
        await renderLoaded(makeProps());

        expect(card('1').getByRole('button', {name: 'Debug'})).toBeInTheDocument();
        expect(card('1').getByRole('button', {name: 'Restart'})).toBeInTheDocument();
        expect(card('1').getByRole('button', {name: 'Lock Deletion'})).toBeInTheDocument();
        expect(card('1').queryByRole('button', {name: 'Unlock Deletion'})).not.toBeInTheDocument();
        expect(card('2').getByRole('button', {name: 'Unlock Deletion'})).toBeInTheDocument();
        expect(card('2').queryByRole('button', {name: 'Lock Deletion'})).not.toBeInTheDocument();
    });

    it('restarts after confirmation without leaving the list', async () => {
        const props = makeProps();
        await renderLoaded(props);

        fireEvent.click(card('1').getByRole('button', {name: 'Restart'}));
        expect(props.actions.restartInstallation).not.toHaveBeenCalled();
        fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', {name: 'Restart'}));

        await waitFor(() => expect(props.actions.restartInstallation).toHaveBeenCalledWith('1'));
        expect(screen.getByTestId('tab-mine')).toBeInTheDocument();
    });

    it('locks deletion directly', async () => {
        const props = makeProps();
        await renderLoaded(props);

        fireEvent.click(card('1').getByRole('button', {name: 'Lock Deletion'}));

        await waitFor(() => expect(props.actions.deletionLockInstallation).toHaveBeenCalledWith('1'));
        expect(screen.getByTestId('tab-mine')).toBeInTheDocument();
    });

    it('unlocks deletion after confirmation', async () => {
        const props = makeProps();
        await renderLoaded(props);

        fireEvent.click(card('2').getByRole('button', {name: 'Unlock Deletion'}));
        fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', {name: 'Unlock'}));

        await waitFor(() => expect(props.actions.deletionUnlockInstallation).toHaveBeenCalledWith('2'));
    });

    it('disables locking when the limit is reached', async () => {
        await renderLoaded(makeProps({maxLockedInstallations: 1}));

        expect(card('1').getByRole('button', {name: 'Lock Deletion'})).toBeDisabled();
    });

    it('shows the error on the card that failed', async () => {
        const props = makeProps();
        props.actions.deletionLockInstallation = jest.fn().mockResolvedValue({error: 'lock limit reached'});
        await renderLoaded(props);

        fireEvent.click(card('1').getByRole('button', {name: 'Lock Deletion'}));

        expect(await card('1').findByText('lock limit reached')).toBeInTheDocument();
        expect(card('2').queryByText('lock limit reached')).not.toBeInTheDocument();
    });

    it('opens the Debug dropdown from the card', async () => {
        const props = makeProps();
        await renderLoaded(props);

        fireEvent.click(card('1').getByRole('button', {name: 'Debug'}));
        fireEvent.click(screen.getByRole('menuitem', {name: 'Get Debug Packet'}));

        expect(props.actions.getDebugPacket).toHaveBeenCalledWith('first');
        expect(screen.getByTestId('tab-mine')).toBeInTheDocument();
    });

    it('limits shared cards by permission', async () => {
        await renderLoaded(makeProps());
        fireEvent.click(screen.getByTestId('tab-shared'));

        expect(card('3').getByRole('button', {name: 'Debug'})).toBeInTheDocument();
        expect(card('3').getByRole('button', {name: 'Restart'})).toBeInTheDocument();
        expect(card('3').queryByRole('button', {name: /Lock Deletion/})).not.toBeInTheDocument();

        expect(card('5').getByRole('button', {name: 'Debug'})).toBeInTheDocument();
        expect(card('5').queryByRole('button', {name: 'Restart'})).not.toBeInTheDocument();
    });

    it('offers no actions for a deleted installation', async () => {
        const props = makeProps();
        props.installs = [{...props.installs[0], State: 'deleted'}];
        await renderLoaded(props);

        expect(card('1').queryByRole('button', {name: 'Debug'})).not.toBeInTheDocument();
        expect(card('1').queryByRole('button', {name: 'Restart'})).not.toBeInTheDocument();
    });
});

describe('SidebarRight card link', () => {
    it('links the domain to the installation in a new tab', async () => {
        await renderLoaded(makeProps());

        const link = screen.getByRole('link', {name: 'Open first.example.com in a new tab'});

        expect(link).toHaveAttribute('href', 'https://first.example.com');
        expect(link).toHaveAttribute('target', '_blank');
        expect(link.getAttribute('rel')).toContain('noopener');
    });

    it('shows no link for an installation without a domain', async () => {
        await renderLoaded(makeProps());

        expect(screen.getAllByRole('link')).toHaveLength(2);
        expect(screen.queryByRole('link', {name: /sleeping/})).not.toBeInTheDocument();
        expect(screen.getByText('No URL!')).toBeInTheDocument();
    });

    it('does not open the detail when the link is clicked', async () => {
        await renderLoaded(makeProps());

        fireEvent.click(screen.getByRole('link', {name: 'Open first.example.com in a new tab'}));

        expect(screen.getByTestId('tab-mine')).toBeInTheDocument();
    });

    it('does not nest the link in a button', async () => {
        await renderLoaded(makeProps());

        const link = screen.getByRole('link', {name: 'Open first.example.com in a new tab'});

        expect(link.closest('button')).toBeNull();
    });
});

describe('SidebarRight search', () => {
    const search = () => screen.getByRole('searchbox', {name: 'Search servers'});

    it('filters the list by name, domain, and version', async () => {
        await renderLoaded(makeProps());

        fireEvent.change(search(), {target: {value: 'FIR'}});
        expect(screen.getByText('first')).toBeInTheDocument();
        expect(screen.queryByText('second')).not.toBeInTheDocument();

        fireEvent.change(search(), {target: {value: 'second.example'}});
        expect(screen.getByText('second')).toBeInTheDocument();
        expect(screen.queryByText('first')).not.toBeInTheDocument();

        fireEvent.change(search(), {target: {value: '9.5.0'}});
        expect(screen.getByText('first')).toBeInTheDocument();
        expect(screen.getByText('second')).toBeInTheDocument();
    });

    it('explains when nothing matches and keeps the add card', async () => {
        await renderLoaded(makeProps());

        fireEvent.change(search(), {target: {value: 'nothing-like-this'}});

        expect(screen.getByText('No matching servers')).toBeInTheDocument();
        expect(screen.getByTestId('add-installation')).toBeInTheDocument();
    });

    it('hides the search bar when the tab has no servers', async () => {
        await renderLoaded(makeProps({installs: [], sharedInstalls: []}));
        expect(screen.queryByRole('searchbox')).not.toBeInTheDocument();

        fireEvent.click(screen.getByTestId('tab-shared'));
        expect(screen.queryByRole('searchbox')).not.toBeInTheDocument();
    });

    it('searches the shared tab', async () => {
        await renderLoaded(makeProps());

        fireEvent.click(screen.getByTestId('tab-shared'));
        fireEvent.change(search(), {target: {value: 'borrowed'}});

        expect(screen.getByText('borrowed')).toBeInTheDocument();
        expect(screen.queryByText('readonly')).not.toBeInTheDocument();
    });

    it('keeps the query after opening a server and going back', async () => {
        await renderLoaded(makeProps());

        fireEvent.change(search(), {target: {value: 'second'}});
        openInstallation('second');
        fireEvent.click(screen.getByRole('button', {name: 'Back'}));

        expect(search()).toHaveValue('second');
        expect(screen.queryByText('first')).not.toBeInTheDocument();
    });

    it('clears the query with the clear button', async () => {
        await renderLoaded(makeProps());

        fireEvent.change(search(), {target: {value: 'second'}});
        fireEvent.click(screen.getByRole('button', {name: 'Clear search'}));

        expect(screen.getByText('first')).toBeInTheDocument();
    });
});

describe('SidebarRight navigation', () => {
    it('goes list > detail > back > list', async () => {
        await renderLoaded(makeProps());

        openInstallation('first');
        expect(screen.getByRole('heading', {name: 'first'})).toBeInTheDocument();
        expect(screen.queryByTestId('tab-mine')).not.toBeInTheDocument();

        fireEvent.click(screen.getByRole('button', {name: 'Back'}));
        expect(screen.getByTestId('tab-mine')).toBeInTheDocument();
    });

    it('keeps the selected tab when coming back from a detail', async () => {
        await renderLoaded(makeProps());

        fireEvent.click(screen.getByTestId('tab-shared'));
        openInstallation('borrowed');
        fireEvent.click(screen.getByRole('button', {name: 'Back'}));

        expect(screen.getByText('borrowed')).toBeInTheDocument();
    });

    it('goes list > add > create > list and creates the installation', async () => {
        const props = makeProps();
        await renderLoaded(props);

        fireEvent.click(screen.getByTestId('add-installation'));
        fireEvent.change(screen.getByLabelText('Name'), {target: {value: 'brand-new'}});
        fireEvent.click(screen.getByRole('button', {name: 'Create'}));

        await waitFor(() => expect(props.actions.createInstallation).toHaveBeenCalled());
        expect(props.actions.createInstallation.mock.calls[0][0]).toMatchObject({name: 'brand-new', version: 'latest', size: 'miniSingleton'});
        expect(await screen.findByTestId('tab-mine')).toBeInTheDocument();
    });

    it('stays on the form and shows the error when creation fails', async () => {
        const props = makeProps();
        props.actions.createInstallation = jest.fn().mockResolvedValue({error: 'Installation name taken already exists'});
        await renderLoaded(props);

        fireEvent.click(screen.getByTestId('add-installation'));
        fireEvent.change(screen.getByLabelText('Name'), {target: {value: 'taken'}});
        fireEvent.click(screen.getByRole('button', {name: 'Create'}));

        expect(await screen.findByText('Installation name taken already exists')).toBeInTheDocument();
        expect(screen.queryByTestId('tab-mine')).not.toBeInTheDocument();
    });

    it('goes detail > edit > apply > detail', async () => {
        const props = makeProps();
        await renderLoaded(props);

        openInstallation('first');
        fireEvent.click(screen.getByRole('button', {name: 'Edit'}));
        fireEvent.change(screen.getByLabelText('Version'), {target: {value: '9.6.0'}});
        fireEvent.click(screen.getByRole('button', {name: 'Apply'}));

        await waitFor(() => expect(props.actions.updateInstallation).toHaveBeenCalledWith({installation_id: '1', version: '9.6.0'}));
        expect(await screen.findByRole('heading', {name: 'first'})).toBeInTheDocument();
    });

    it('cancel in the form returns to the previous view', async () => {
        await renderLoaded(makeProps());

        fireEvent.click(screen.getByTestId('add-installation'));
        fireEvent.click(screen.getByRole('button', {name: 'Cancel'}));

        expect(screen.getByTestId('tab-mine')).toBeInTheDocument();
    });

    it('shows a not-found state when the open installation disappears', async () => {
        const props = makeProps();
        const {rerender} = await renderLoaded(props);
        openInstallation('first');

        rerender(
            <SidebarRight
                {...props}
                installs={props.installs.filter((install) => install.ID !== '1')}
            />,
        );

        expect(screen.getByText('Server not found')).toBeInTheDocument();
        fireEvent.click(screen.getByRole('button', {name: 'Back to list'}));
        expect(screen.getByTestId('tab-mine')).toBeInTheDocument();
    });
});

describe('SidebarRight detail actions', () => {
    it('opens the server URL', async () => {
        const open = jest.spyOn(window, 'open').mockImplementation(() => null);
        await renderLoaded(makeProps());

        openInstallation('first');
        fireEvent.click(screen.getByRole('button', {name: 'Open'}));

        expect(open).toHaveBeenCalledWith('https://first.example.com', '_blank');
        open.mockRestore();
    });

    it('disables Open when the installation has no DNS record', async () => {
        await renderLoaded(makeProps());

        openInstallation('sleeping');

        expect(screen.getByRole('button', {name: 'Open'})).toBeDisabled();
    });

    it('opens the Debug dropdown with its options', async () => {
        const props = makeProps();
        const open = jest.spyOn(window, 'open').mockImplementation(() => null);
        await renderLoaded(props);

        openInstallation('first');
        const trigger = screen.getByRole('button', {name: 'Debug'});
        expect(trigger).toHaveAttribute('aria-haspopup', 'menu');
        fireEvent.click(trigger);

        const items = screen.getAllByRole('menuitem').map((item) => item.textContent);
        expect(items).toEqual(['Installation Logs', 'Provisioner Logs', 'Get Debug Packet']);

        fireEvent.click(screen.getByRole('menuitem', {name: 'Installation Logs'}));
        expect(open).toHaveBeenCalledWith('https://logs.example.com/installation', '_blank');

        fireEvent.click(screen.getByRole('button', {name: 'Debug'}));
        fireEvent.click(screen.getByRole('menuitem', {name: 'Get Debug Packet'}));
        expect(props.actions.getDebugPacket).toHaveBeenCalledWith('first');
        open.mockRestore();
    });

    it('restarts after confirmation', async () => {
        const props = makeProps();
        await renderLoaded(props);

        openInstallation('first');
        fireEvent.click(screen.getByRole('button', {name: 'Restart'}));
        expect(props.actions.restartInstallation).not.toHaveBeenCalled();
        fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', {name: 'Restart'}));

        await waitFor(() => expect(props.actions.restartInstallation).toHaveBeenCalledWith('1'));
    });

    it('hibernates a stable installation and wakes a hibernating one', async () => {
        const props = makeProps();
        await renderLoaded(props);

        openInstallation('first');
        expect(screen.queryByRole('button', {name: 'Wake up'})).not.toBeInTheDocument();
        fireEvent.click(screen.getByRole('button', {name: 'Hibernate'}));
        fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', {name: 'Hibernate'}));
        await waitFor(() => expect(props.actions.hibernateInstallation).toHaveBeenCalledWith('1'));

        fireEvent.click(screen.getByRole('button', {name: 'Back'}));
        openInstallation('sleeping');
        expect(screen.queryByRole('button', {name: 'Hibernate'})).not.toBeInTheDocument();
        fireEvent.click(screen.getByRole('button', {name: 'Wake up'}));
        await waitFor(() => expect(props.actions.wakeInstallation).toHaveBeenCalledWith('4'));
    });

    it('locks deletion, and unlocks it after confirmation', async () => {
        const props = makeProps();
        await renderLoaded(props);

        openInstallation('first');
        fireEvent.click(screen.getByRole('button', {name: 'Lock Deletion'}));
        await waitFor(() => expect(props.actions.deletionLockInstallation).toHaveBeenCalledWith('1'));

        fireEvent.click(screen.getByRole('button', {name: 'Back'}));
        openInstallation('second');
        fireEvent.click(screen.getByRole('button', {name: 'Unlock Deletion'}));
        fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', {name: 'Unlock'}));
        await waitFor(() => expect(props.actions.deletionUnlockInstallation).toHaveBeenCalledWith('2'));
    });

    it('disables locking when the limit is reached', async () => {
        await renderLoaded(makeProps({maxLockedInstallations: 1}));

        openInstallation('first');

        expect(screen.getByRole('button', {name: 'Lock Deletion'})).toBeDisabled();
    });

    it('shows the action error returned by the server', async () => {
        const props = makeProps();
        props.actions.deletionLockInstallation = jest.fn().mockResolvedValue({error: 'you may only have at most 2 installations locked'});
        await renderLoaded(props);

        openInstallation('first');
        fireEvent.click(screen.getByRole('button', {name: 'Lock Deletion'}));

        expect(await screen.findByText('you may only have at most 2 installations locked')).toBeInTheDocument();
    });

    it('requires typing the name before deleting, then returns to the list', async () => {
        const props = makeProps();
        await renderLoaded(props);

        openInstallation('first');
        fireEvent.click(screen.getByRole('button', {name: 'Delete'}));
        const dialog = screen.getByRole('dialog');
        const confirm = within(dialog).getByRole('button', {name: 'Delete'});
        expect(confirm).toBeDisabled();

        fireEvent.change(within(dialog).getByLabelText('Type first to confirm'), {target: {value: 'first'}});
        expect(confirm).toBeEnabled();
        fireEvent.click(confirm);

        await waitFor(() => expect(props.actions.deleteInstallation).toHaveBeenCalledWith('1', 'first'));
        expect(await screen.findByTestId('tab-mine')).toBeInTheDocument();
    });

    it('does not offer delete while deletion is locked', async () => {
        await renderLoaded(makeProps());

        openInstallation('second');

        expect(screen.getByRole('button', {name: 'Delete'})).toBeDisabled();
    });

    it('updates sharing with the switches', async () => {
        const props = makeProps();
        await renderLoaded(props);

        openInstallation('first');
        expect(screen.getByRole('switch', {name: /Allow shared updates/})).toBeDisabled();
        fireEvent.click(screen.getByRole('switch', {name: /Share with other plugin users/}));

        await waitFor(() => expect(props.actions.shareInstallation).toHaveBeenCalledWith('1', true, false));
    });

    it('limits a shared installation with updates to edit and restart', async () => {
        await renderLoaded(makeProps());

        fireEvent.click(screen.getByTestId('tab-shared'));
        openInstallation('borrowed');

        expect(screen.getByRole('button', {name: 'Edit'})).toBeInTheDocument();
        expect(screen.getByRole('button', {name: 'Restart'})).toBeInTheDocument();
        expect(screen.queryByRole('button', {name: 'Delete'})).not.toBeInTheDocument();
        expect(screen.queryByRole('button', {name: 'Hibernate'})).not.toBeInTheDocument();
        expect(screen.queryByRole('switch')).not.toBeInTheDocument();
    });

    it('gives read-only access to a shared installation without updates', async () => {
        await renderLoaded(makeProps());

        fireEvent.click(screen.getByTestId('tab-shared'));
        openInstallation('readonly');

        expect(screen.queryByRole('button', {name: 'Edit'})).not.toBeInTheDocument();
        expect(screen.queryByRole('button', {name: 'Restart'})).not.toBeInTheDocument();
    });
});

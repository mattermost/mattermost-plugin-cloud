import React from 'react';
import {render, screen, fireEvent} from '@testing-library/react';

import SidebarRight from './sidebar_right';

describe('SidebarRight', () => {
    const props = {
        id: 'test-id',
        installs: [
            {
                ID: '1',
                Name: 'Test Installation 1',
                State: 'stable',
                DNSRecords: [{DomainName: 'test1.example.com'}],
                Image: 'test-image',
                Tag: '',
                Database: 'test-db',
                Filestore: 'test-filestore',
                Size: 'test-size',
                InstallationLogsURL: 'https://test1.example.com/logs',
                ProvisionerLogsURL: 'https://test1.example.com/provisioner-logs',
                DeletionLocked: false,
            },
            {
                ID: '2',
                Name: 'Test Installation 2',
                State: 'in progress',
                DNSRecords: [{DomainName: 'test2.example.com'}],
                Image: 'test-image',
                Tag: '',
                Database: 'test-db',
                Filestore: 'test-filestore',
                Size: 'test-size',
                InstallationLogsURL: 'https://test2.example.com/logs',
                ProvisionerLogsURL: 'https://test2.example.com/provisioner-logs',
                DeletionLocked: true,
            },
        ],
        serverError: '',
        deletionLockedInstallationId: null,
        maxLockedInstallations: 1,
        actions: {
            setVisible: jest.fn(),
            telemetry: jest.fn(),
            getCloudUserData: jest.fn(),
            deletionLockInstallation: jest.fn(),
            deletionUnlockInstallation: jest.fn(),
            getPluginConfiguration: jest.fn(),
        },
    };

    it('renders a list of installations', () => {
        render(<SidebarRight {...props}/>);

        const installation1 = screen.getByText('Test Installation 1');
        const installation2 = screen.getByText('Test Installation 2');

        expect(installation1).toBeInTheDocument();
        expect(installation2).toBeInTheDocument();
    });

    it('displays a message when there are no installations', () => {
        const propsWithNoInstalls = {...props, installs: []};
        render(<SidebarRight {...propsWithNoInstalls}/>);

        const message = screen.getByText('There are no installations, use the /cloud create command to add an installation.');

        expect(message).toBeInTheDocument();
    });

    it('displays a message when there is a server error', () => {
        const propsWithServerError = {...props, serverError: 'Test server error'};
        render(<SidebarRight {...propsWithServerError}/>);

        const message = screen.getByText('Received a server error');
        const error = screen.getByText('Test server error');

        expect(message).toBeInTheDocument();
        expect(error).toBeInTheDocument();
    });

    it('calls the setVisible action on mount', () => {
        render(<SidebarRight {...props}/>);

        expect(props.actions.setVisible).toHaveBeenCalledWith(true);
    });

    it('calls the setVisible action on unmount', () => {
        const {unmount} = render(<SidebarRight {...props}/>);

        unmount();

        expect(props.actions.setVisible).toHaveBeenCalledWith(false);
    });

    it('calls the getCloudUserData and getPluginConfiguration actions on mount', () => {
        render(<SidebarRight {...props}/>);

        expect(props.actions.getCloudUserData).toHaveBeenCalledWith('test-id');
        expect(props.actions.getPluginConfiguration).toHaveBeenCalled();
    });

    it('calls the deletionLockInstallation action when the lock deletion button is clicked', () => {
        const newProps = props;

        // Need to bump to 2 allowed locked installations so the lock button isn't disabled
        newProps.maxLockedInstallations = 2;
        render(<SidebarRight {...newProps}/>);

        const lockButton = screen.getByText('Lock Deletion');
        fireEvent.click(lockButton);

        expect(props.actions.deletionLockInstallation).toHaveBeenCalledWith('1');
    });

    it('calls the deletionUnlockInstallation action when the unlock deletion button is clicked', () => {
        render(<SidebarRight {...props}/>);

        const unlockButton = screen.getByText('Unlock Deletion');
        fireEvent.click(unlockButton);

        expect(props.actions.deletionUnlockInstallation).toHaveBeenCalledWith('2');
    });

    it('disables the lock deletion button when the maximum number of locked installations is reached', () => {
        const newProps = props;
        newProps.maxLockedInstallations = 1;
        render(<SidebarRight {...newProps}/>);

        const lockButton = screen.getByText('Lock Deletion');
        fireEvent.click(lockButton);

        expect(props.actions.deletionLockInstallation).toHaveBeenCalledWith('1');

        const lockButton2 = screen.getByText('Lock Deletion');
        fireEvent.click(lockButton2);

        expect(props.actions.deletionLockInstallation).not.toHaveBeenCalledWith('2');
        expect(lockButton2).toBeDisabled();
    });
});
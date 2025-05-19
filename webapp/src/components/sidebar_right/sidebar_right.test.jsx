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
        sharedInstalls: [
            {
                ID: '3',
                Name: 'Test Installation 3',
                State: 'stable',
                DNSRecords: [{DomainName: 'test3.example.com'}],
                Image: 'test-image',
                Tag: '',
                Database: 'test-db',
                Filestore: 'test-filestore',
                Size: 'test-size',
                InstallationLogsURL: 'https://test3.example.com/logs',
                ProvisionerLogsURL: 'https://test3.example.com/provisioner-logs',
                DeletionLocked: false,
            },
        ],
        serverError: '',
        deletionLockedInstallationId: null,
        maxLockedInstallations: 1,
        actions: {
            setVisible: jest.fn(),
            telemetry: jest.fn(),
            getCloudUserData: jest.fn(),
            getSharedInstalls: jest.fn(),
            restartInstallation: jest.fn(),
            deletionLockInstallation: jest.fn(),
            deletionUnlockInstallation: jest.fn(),
            getPluginConfiguration: jest.fn(),
            getDebugPacket: jest.fn(),
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

        const message = screen.getByText('There are no installations. Use the `/cloud create` command to add an installation.');

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

    it('calls the getCloudUserData, getSharedInstalls, and getPluginConfiguration actions on mount', () => {
        render(<SidebarRight {...props}/>);

        expect(props.actions.getCloudUserData).toHaveBeenCalledWith('test-id');
        expect(props.actions.getSharedInstalls).toHaveBeenCalled();
        expect(props.actions.getPluginConfiguration).toHaveBeenCalled();
    });

    it('calls the restartInstallation action when the restart button is clicked', () => {
        render(<SidebarRight {...props}/>);

        const restartButton = screen.getByTestId('restart-1');
        fireEvent.click(restartButton);

        const confirmButton = screen.getByText('Confirm');
        fireEvent.click(confirmButton);

        expect(props.actions.restartInstallation).toHaveBeenCalledWith('Test Installation 1');
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

        const confirmButton = screen.getByText('Confirm');
        fireEvent.click(confirmButton);

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

    it('renders a list of shared installations', () => {
        render(<SidebarRight {...props}/>);

        const sharedServerButton = screen.getByText('Shared');
        fireEvent.click(sharedServerButton);

        const installation3 = screen.getByText('Test Installation 3');

        expect(installation3).toBeInTheDocument();
    });

    it('displays a message when there are no shared installations', () => {
        const propsWithNoSharedInstalls = {...props, sharedInstalls: []};
        render(<SidebarRight {...propsWithNoSharedInstalls}/>);

        const sharedServerButton = screen.getByText('Shared');
        fireEvent.click(sharedServerButton);

        const message = screen.getByText('There are no shared installations. Use the `/cloud share` command to share one of yours with other plugin users.');

        expect(message).toBeInTheDocument();
    });

    // Create a timestamp for 2 days in the future
    const futureDateIn2Days = new Date();
    futureDateIn2Days.setDate(futureDateIn2Days.getDate() + 2);

    // Create a timestamp for 12 hours in the future
    const futureDateIn12Hours = new Date();
    futureDateIn12Hours.setHours(futureDateIn12Hours.getHours() + 12);

    it('displays scheduled deletion time when ScheduledDeletionTime is set', () => {
        const propsWithScheduledDeletion = {
            ...props,
            installs: [
                {
                    ...props.installs[0],
                    ScheduledDeletionTime: futureDateIn2Days.getTime(),
                },
            ],
        };

        render(<SidebarRight {...propsWithScheduledDeletion}/>);

        // Check that the "Deleting:" label is displayed
        const deletingLabel = screen.getByText('Deleting:');
        expect(deletingLabel).toBeInTheDocument();

        // Check that a time indication is displayed
        const dayText = screen.getByText((content) =>
            content.includes('day') || content.includes('days'), {exact: false},
        );
        expect(dayText).toBeInTheDocument();
    });

    it('does not display scheduled deletion time when ScheduledDeletionTime is 0', () => {
        const propsWithNoScheduledDeletion = {
            ...props,
            installs: [
                {
                    ...props.installs[0],
                    ScheduledDeletionTime: 0,
                },
            ],
        };

        render(<SidebarRight {...propsWithNoScheduledDeletion}/>);

        // Check that the "Deleting:" label is not displayed
        const deletingLabel = screen.queryByText('Deleting:');
        expect(deletingLabel).not.toBeInTheDocument();
    });

    it('does not display scheduled deletion time when DeletionLocked is true', () => {
        const propsWithLockedDeletion = {
            ...props,
            installs: [
                {
                    ...props.installs[0],
                    ScheduledDeletionTime: futureDateIn2Days.getTime(),
                    DeletionLocked: true,
                },
            ],
        };

        render(<SidebarRight {...propsWithLockedDeletion}/>);

        // Check that the "Deleting:" label is not displayed
        const deletingLabel = screen.queryByText('Deleting:');
        expect(deletingLabel).not.toBeInTheDocument();
    });

    it('formats deletion time correctly for days in the future', () => {
        // Mock the formatDeletionTime method
        const sidebarRightInstance = new SidebarRight({...props});

        // 2 days in the future
        const formattedTime = sidebarRightInstance.formatDeletionTime(futureDateIn2Days.getTime());

        // Should show days
        expect(formattedTime).toContain('day');
    });

    it('formats deletion time correctly for hours in the future', () => {
        // Mock the formatDeletionTime method
        const sidebarRightInstance = new SidebarRight({...props});

        // 12 hours in the future
        const formattedTime = sidebarRightInstance.formatDeletionTime(futureDateIn12Hours.getTime());

        // Should show hours
        expect(formattedTime).toContain('hour');
    });

    it('uses normal styling for installations with deletion > 1 day away', () => {
        // Mock the getTimeRemainingStyle method
        const sidebarRightInstance = new SidebarRight({...props});
        const style = sidebarRightInstance.getTimeRemainingStyle(futureDateIn2Days.getTime());

        // Should return normalDeletionTime style
        expect(style).toEqual(expect.objectContaining({
            color: expect.any(String),
            fontWeight: expect.any(String),
        }));
    });

    it('uses urgent styling for installations with deletion < 1 day away', () => {
        // Mock the getTimeRemainingStyle method
        const sidebarRightInstance = new SidebarRight({...props});
        const style = sidebarRightInstance.getTimeRemainingStyle(futureDateIn12Hours.getTime());

        // Should return urgentDeletionTime style
        expect(style).toEqual(expect.objectContaining({
            color: expect.any(String),
            fontWeight: expect.any(String),
        }));
    });
});

import React from 'react';
import {render, screen} from '@testing-library/react';

import UserAttribute from './user_attribute';

describe('UserAttribute', () => {
    it('renders null when installs array is empty', () => {
        const props = {
            id: '123',
            installs: [],
            serverError: '',
            actions: {
                getCloudUserData: jest.fn(),
            },
        };

        render(<UserAttribute {...props}/>);

        expect(screen.queryByText('Cloud Servers')).toBeNull();
    });

    it('renders null when serverError is not empty', () => {
        const props = {
            id: '123',
            installs: [],
            serverError: 'Error message',
            actions: {
                getCloudUserData: jest.fn(),
            },
        };

        render(<UserAttribute {...props}/>);

        expect(screen.queryByText('Cloud Servers')).toBeNull();
    });

    it('renders the list of installations', () => {
        const props = {
            id: '123',
            installs: [
                {
                    ID: '1',
                    Name: 'Test Installation 1',
                    DNSRecords: [{DomainName: 'example.com'}],
                },
                {
                    ID: '2',
                    Name: 'Test Installation 2',
                    DNSRecords: [],
                },
            ],
            serverError: '',
            actions: {
                getCloudUserData: jest.fn(),
            },
        };

        render(<UserAttribute {...props}/>);

        expect(screen.getByText('Cloud Servers')).toBeInTheDocument();
        expect(screen.getByText('Test Installation 1')).toBeInTheDocument();
        expect(screen.getByText('Test Installation 2 (No URL)')).toBeInTheDocument();
    });
});
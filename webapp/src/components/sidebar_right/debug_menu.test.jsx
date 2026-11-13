import React from 'react';
import {render, screen, fireEvent, waitFor} from '@testing-library/react';

import DebugMenu from './debug_menu';

describe('DebugMenu', () => {
    const onLogs = jest.fn();
    const onPacket = jest.fn();
    const items = [
        {key: 'logs', label: 'Installation logs', onClick: onLogs},
        {key: 'packet', label: 'Get debug packet', onClick: onPacket},
    ];

    beforeEach(() => {
        jest.clearAllMocks();
    });

    it('hides the menu items until the trigger is clicked', () => {
        render(
            <DebugMenu
                title='Debug'
                items={items}
            />,
        );

        expect(screen.queryByRole('menuitem')).toBeNull();
        expect(screen.getByRole('button', {name: 'Debug'})).toHaveAttribute('aria-expanded', 'false');
    });

    it('shows the menu items when the trigger is clicked', () => {
        render(
            <DebugMenu
                title='Debug'
                items={items}
            />,
        );

        fireEvent.click(screen.getByRole('button', {name: 'Debug'}));

        expect(screen.getAllByRole('menuitem')).toHaveLength(2);
        expect(screen.getByRole('button', {name: 'Debug'})).toHaveAttribute('aria-expanded', 'true');
    });

    it('runs the item action and closes the menu', async () => {
        render(
            <DebugMenu
                title='Debug'
                items={items}
            />,
        );

        fireEvent.click(screen.getByRole('button', {name: 'Debug'}));
        fireEvent.click(screen.getByRole('menuitem', {name: 'Get debug packet'}));

        expect(onPacket).toHaveBeenCalledTimes(1);
        await waitFor(() => expect(screen.queryAllByRole('menuitem')).toHaveLength(0));
    });

    it('closes the menu when Escape is pressed', async () => {
        render(
            <DebugMenu
                title='Debug'
                items={items}
            />,
        );

        fireEvent.click(screen.getByRole('button', {name: 'Debug'}));
        fireEvent.keyDown(document, {key: 'Escape'});

        await waitFor(() => expect(screen.queryAllByRole('menuitem')).toHaveLength(0));
        expect(onLogs).not.toHaveBeenCalled();
    });

    it('closes the menu when the user clicks outside', async () => {
        render(
            <div>
                <DebugMenu
                    title='Debug'
                    items={items}
                />
                <p>{'outside'}</p>
            </div>,
        );

        fireEvent.click(screen.getByRole('button', {name: 'Debug'}));
        fireEvent.mouseDown(screen.getByText('outside'));
        fireEvent.click(screen.getByText('outside'));

        await waitFor(() => expect(screen.queryAllByRole('menuitem')).toHaveLength(0));
    });
});

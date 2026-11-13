import React from 'react';
import {render, screen, fireEvent} from '@testing-library/react';

import DebugMenu from './debug_menu';

describe('DebugMenu', () => {
    const onLogs = jest.fn();
    const onPacket = jest.fn();
    const items = [
        {key: 'logs', label: 'Installation Logs', onClick: onLogs},
        {key: 'packet', label: 'Get Debug Packet', onClick: onPacket},
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

    it('moves focus to the first item when the menu opens', () => {
        render(
            <DebugMenu
                title='Debug'
                items={items}
            />,
        );

        fireEvent.click(screen.getByRole('button', {name: 'Debug'}));

        expect(screen.getByRole('menuitem', {name: 'Installation Logs'})).toHaveFocus();
    });

    it('runs the item action and closes the menu', () => {
        render(
            <DebugMenu
                title='Debug'
                items={items}
            />,
        );

        fireEvent.click(screen.getByRole('button', {name: 'Debug'}));
        fireEvent.click(screen.getByRole('menuitem', {name: 'Get Debug Packet'}));

        expect(onPacket).toHaveBeenCalledTimes(1);
        expect(screen.queryByRole('menuitem')).toBeNull();
    });

    it('closes the menu when Escape is pressed', () => {
        render(
            <DebugMenu
                title='Debug'
                items={items}
            />,
        );

        fireEvent.click(screen.getByRole('button', {name: 'Debug'}));
        fireEvent.keyDown(document, {key: 'Escape'});

        expect(screen.queryByRole('menuitem')).toBeNull();
        expect(onLogs).not.toHaveBeenCalled();
    });

    it('closes the menu when the user clicks outside', () => {
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

        expect(screen.queryByRole('menuitem')).toBeNull();
    });
});

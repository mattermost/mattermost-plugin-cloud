import React from 'react';
import {render, screen} from '@testing-library/react';

import ChannelHeaderButton from './channel_header_button';

describe('ChannelHeaderButton', () => {
    it('renders the button without highlighting', () => {
        const props = {
            shouldHighlight: false,
        };

        render(<ChannelHeaderButton {...props}/>);

        const button = screen.getByRole('button');
        expect(button).toBeInTheDocument();
        expect(button).toHaveClass('icon fa fa-cloud');
        expect(button).not.toHaveClass('todo-plugin-icon--active');
    });

    it('renders the button with highlighting', () => {
        const props = {
            shouldHighlight: true,
        };

        render(<ChannelHeaderButton {...props}/>);

        const button = screen.getByRole('button');
        expect(button).toBeInTheDocument();
        expect(button).toHaveClass('icon fa fa-cloud');
        expect(button).toHaveClass('todo-plugin-icon--active');
    });
});
import React from 'react';
import {render, screen, fireEvent} from '@testing-library/react';

import ConfirmationModal from './confirmation_modal';

describe('DeletionUnlockConfirmationModal', () => {
    const mockOnConfirm = jest.fn();
    const mockOnCancel = jest.fn();

    beforeEach(() => {
        jest.clearAllMocks();
    });

    it('renders the modal with the correct title and message', () => {
        const title = 'Title Message';
        const bodyText = 'Scary message about what could go wrong';

        const props = {
            title,
            bodyText,
            visible: true,
            onConfirm: mockOnConfirm,
            onCancel: mockOnCancel,
        };

        render(<ConfirmationModal {...props}/>);

        expect(screen.getByText(title)).toBeInTheDocument();
        expect(screen.getByText(bodyText)).toBeInTheDocument();
    });

    it('calls the onConfirm function when "Confirm" button is clicked', () => {
        const props = {
            title: 'title',
            bodyText: 'text',
            visible: true,
            onConfirm: mockOnConfirm,
            onCancel: mockOnCancel,
        };

        render(<ConfirmationModal {...props}/>);

        const confirmButton = screen.getByText('Confirm');
        fireEvent.click(confirmButton);

        expect(mockOnConfirm).toHaveBeenCalledTimes(1);
    });

    it('calls the onCancel function when "Cancel" button is clicked', () => {
        const props = {
            title: 'title',
            bodyText: 'text',
            visible: true,
            onConfirm: mockOnConfirm,
            onCancel: mockOnCancel,
        };

        render(<ConfirmationModal {...props}/>);

        const cancelButton = screen.getByText('Cancel');
        fireEvent.click(cancelButton);

        expect(mockOnCancel).toHaveBeenCalledTimes(1);
    });

    it('does not render the modal when visible is false', () => {
        const title = 'Title Message';
        const bodyText = 'Scary message about what could go wrong';

        const props = {
            title,
            bodyText,
            visible: false,
            onConfirm: mockOnConfirm,
            onCancel: mockOnCancel,
        };

        render(<ConfirmationModal {...props}/>);

        expect(screen.queryByText(title)).toBeNull();
        expect(screen.queryByText(bodyText)).toBeNull();
    });
});

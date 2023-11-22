import React from 'react';
import {render, screen, fireEvent} from '@testing-library/react';

import DeletionUnlockConfirmationModal from './deletion_unlock_confirmation_modal';

describe('DeletionUnlockConfirmationModal', () => {
    const mockOnConfirm = jest.fn();
    const mockOnCancel = jest.fn();

    beforeEach(() => {
        jest.clearAllMocks();
    });

    it('renders the modal with the correct title and message', () => {
        const props = {
            visible: true,
            onConfirm: mockOnConfirm,
            onCancel: mockOnCancel,
        };

        render(<DeletionUnlockConfirmationModal {...props}/>);

        expect(screen.getByText('Remove deletion lock?')).toBeInTheDocument();
        expect(
            screen.getByText(
                'Are you sure you want to remove the deletion lock? Doing so will add this installation back into the clean up pool, meaning it can be deleted.',
            ),
        ).toBeInTheDocument();
    });

    it('calls the onConfirm function when "Remove Lock" button is clicked', () => {
        const props = {
            visible: true,
            onConfirm: mockOnConfirm,
            onCancel: mockOnCancel,
        };

        render(<DeletionUnlockConfirmationModal {...props}/>);

        const removeLockButton = screen.getByText('Remove Lock');
        fireEvent.click(removeLockButton);

        expect(mockOnConfirm).toHaveBeenCalledTimes(1);
    });

    it('calls the onCancel function when "Cancel" button is clicked', () => {
        const props = {
            visible: true,
            onConfirm: mockOnConfirm,
            onCancel: mockOnCancel,
        };

        render(<DeletionUnlockConfirmationModal {...props}/>);

        const cancelButton = screen.getByText('Cancel');
        fireEvent.click(cancelButton);

        expect(mockOnCancel).toHaveBeenCalledTimes(1);
    });

    it('does not render the modal when visible is false', () => {
        const props = {
            visible: false,
            onConfirm: mockOnConfirm,
            onCancel: mockOnCancel,
        };

        render(<DeletionUnlockConfirmationModal {...props}/>);

        expect(screen.queryByText('Remove deletion lock?')).toBeNull();
        expect(
            screen.queryByText(
                'Are you sure you want to remove the deletion lock? Doing so will add this installation back into the clean up pool, meaning it can be deleted.',
            ),
        ).toBeNull();
    });
});
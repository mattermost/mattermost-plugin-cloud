import React from 'react';
import { Button, Modal } from 'react-bootstrap';
import './deletion_unlock_confirmation_modal.scss'


export default function DeletionUnlockConfirmationModal({visible, onConfirm, onCancel}) {
    return (
        <Modal
            show={visible}
            onHide = {onCancel}
            onCancel={onCancel}
            className={'CloudPluginUnlockDeletionConfirmationModal'}
        >
            <Modal.Header closeButton={false}>
                <Modal.Title>Remove deletion lock?</Modal.Title>
            </Modal.Header>
            <Modal.Body>
                <p>
                    Are you sure you want to remove the deletion lock? Doing so will add this installation back into the clean up pool, meaning it can be deleted.
                </p>
            </Modal.Body>
            <Modal.Footer>
                <Button
                    type="button"
                    bsStyle="tertiary"
                    onClick={onCancel}
                >
                    Cancel
                </Button>
                <Button
                    type="button"
                    bsStyle="danger"
                    onClick={onConfirm}
                >
                    Remove Lock
                </Button>
            </Modal.Footer>
        </Modal>
    )
}
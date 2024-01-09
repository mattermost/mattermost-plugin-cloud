import React from 'react';
import {Button, Modal} from 'react-bootstrap';
import PropTypes from 'prop-types';
import './confirmation_modal.scss';

function ConfirmationModal({title, bodyText, visible, onConfirm, onCancel}) {
    return (
        <Modal
            title={title}
            bodyText={bodyText}
            show={visible}
            onHide={onCancel}
            onCancel={onCancel}
            className={'CloudPluginConfirmationModal'}
        >
            <Modal.Header closeButton={false}>
                <Modal.Title>{title}</Modal.Title>
            </Modal.Header>
            <Modal.Body>
                <p>{bodyText}</p>
            </Modal.Body>
            <Modal.Footer>
                <Button
                    type='button'
                    bsStyle='tertiary'
                    onClick={onCancel}
                >
                    Cancel
                </Button>
                <Button
                    type='button'
                    bsStyle='danger'
                    onClick={onConfirm}
                >
                    Confirm
                </Button>
            </Modal.Footer>
        </Modal>
    );
}

ConfirmationModal.propTypes = {
    title: PropTypes.string.isRequired,
    bodyText: PropTypes.string.isRequired,
    visible: PropTypes.bool.isRequired,
    onConfirm: PropTypes.func.isRequired,
    onCancel: PropTypes.func.isRequired,
};

export default ConfirmationModal;

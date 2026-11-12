import React from 'react';
import PropTypes from 'prop-types';
import {Button, Modal, ModalOverlay} from '@mattermost/compass-ui';

function ConfirmationModal({title, bodyText, visible, onConfirm, onCancel}) {
    return (
        <ModalOverlay
            open={visible}
            onClose={onCancel}
        >
            <Modal
                size='small'
                title={title}
                onClose={onCancel}
                footer={(
                    <>
                        <Button
                            type='button'
                            emphasis='tertiary'
                            onClick={onCancel}
                        >
                            {'Cancel'}
                        </Button>
                        <Button
                            type='button'
                            emphasis='primary'
                            destructive={true}
                            onClick={onConfirm}
                        >
                            {'Confirm'}
                        </Button>
                    </>
                )}
            >
                <p>{bodyText}</p>
            </Modal>
        </ModalOverlay>
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

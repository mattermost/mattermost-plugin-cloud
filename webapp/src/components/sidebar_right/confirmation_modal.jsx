import React, {useEffect, useState} from 'react';
import PropTypes from 'prop-types';
import {Button, Modal, ModalOverlay, TextInput} from '@mattermost/compass-ui';

function ConfirmationModal({title, bodyText, visible, onConfirm, onCancel, confirmLabel = 'Confirm', requiredText = ''}) {
    const [typedText, setTypedText] = useState('');
    useEffect(() => {
        if (!visible) {
            setTypedText('');
        }
    }, [visible]);

    const confirmDisabled = Boolean(requiredText) && typedText !== requiredText;

    return (
        <ModalOverlay
            open={visible}
            onClose={onCancel}
        >
            <Modal
                open={visible}
                size='small'
                title={title}
                headerDivider={false}
                footerDivider={false}
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
                            disabled={confirmDisabled}
                            onClick={onConfirm}
                        >
                            {confirmLabel}
                        </Button>
                    </>
                )}
            >
                <p>{bodyText}</p>
                {requiredText && (
                    <TextInput
                        label={`Type ${requiredText} to confirm`}
                        value={typedText}
                        onChange={(e) => setTypedText(e.target.value)}
                    />
                )}
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
    confirmLabel: PropTypes.string,
    requiredText: PropTypes.string,
};

export default ConfirmationModal;

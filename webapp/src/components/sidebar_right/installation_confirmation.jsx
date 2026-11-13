import React from 'react';
import PropTypes from 'prop-types';

import ConfirmationModal from './confirmation_modal';
import {CONFIRMATIONS} from './use_installation_actions';

export default function InstallationConfirmation({installation, confirmation, onConfirm, onCancel}) {
    const copy = CONFIRMATIONS[confirmation];
    if (!copy) {
        return null;
    }

    return (
        <ConfirmationModal
            title={copy.title}
            bodyText={copy.bodyText}
            confirmLabel={copy.confirmLabel}
            requiredText={confirmation === 'delete' ? installation.Name : ''}
            visible={true}
            onConfirm={() => onConfirm(confirmation)}
            onCancel={onCancel}
        />
    );
}

InstallationConfirmation.propTypes = {
    installation: PropTypes.object.isRequired,
    confirmation: PropTypes.string.isRequired,
    onConfirm: PropTypes.func.isRequired,
    onCancel: PropTypes.func.isRequired,
};

import React, {useRef} from 'react';
import PropTypes from 'prop-types';

import ConfirmationModal from './confirmation_modal';
import {CONFIRMATIONS} from './use_installation_actions';

// The modal stays mounted so it can animate out. The last copy stays in place while it closes.
export default function InstallationConfirmation({installation, confirmation, onConfirm, onCancel}) {
    const lastConfirmation = useRef(confirmation);
    if (confirmation) {
        lastConfirmation.current = confirmation;
    }

    const current = lastConfirmation.current;
    const copy = CONFIRMATIONS[current];
    if (!copy) {
        return null;
    }

    return (
        <ConfirmationModal
            title={copy.title}
            bodyText={copy.bodyText}
            confirmLabel={copy.confirmLabel}
            requiredText={current === 'delete' ? installation.Name : ''}
            visible={Boolean(confirmation)}
            onConfirm={() => onConfirm(current)}
            onCancel={onCancel}
        />
    );
}

InstallationConfirmation.propTypes = {
    installation: PropTypes.object.isRequired,
    confirmation: PropTypes.string,
    onConfirm: PropTypes.func.isRequired,
    onCancel: PropTypes.func.isRequired,
};

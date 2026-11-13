import {useState} from 'react';

export const CONFIRMATIONS = {
    restart: {
        title: 'Restart installation servers?',
        bodyText: 'Are you sure you want to restart the mattermost server instances? Doing so force new servers to be created while removing the old ones.',
        confirmLabel: 'Restart',
    },
    unlock: {
        title: 'Remove deletion lock?',
        bodyText: 'Are you sure you want to remove the deletion lock? Doing so will add this installation back into the clean up pool, meaning it can be deleted.',
        confirmLabel: 'Unlock',
    },
    hibernate: {
        title: 'Hibernate installation?',
        bodyText: 'The server stops until you wake it up again. Users cannot reach it while it hibernates.',
        confirmLabel: 'Hibernate',
    },
    delete: {
        title: 'Delete installation?',
        bodyText: 'This action deletes the installation. The name stays reserved for 24 hours.',
        confirmLabel: 'Delete',
    },
};

export function getDebugMenuItems(installation, actions) {
    return [
        {key: 'installation-logs', onClick: () => window.open(installation.InstallationLogsURL, '_blank'), label: 'Installation logs'},
        {key: 'provisioner-logs', onClick: () => window.open(installation.ProvisionerLogsURL, '_blank'), label: 'Provisioner logs'},
        {key: 'debug-packet', onClick: () => actions.getDebugPacket(installation.Name), label: 'Get debug packet'},
    ];
}

// Shared by the list card and the detail view. Holds the busy action, the last error, and the open confirmation.
export default function useInstallationActions({installation, actions, onDeleted}) {
    const [busyAction, setBusyAction] = useState('');
    const [error, setError] = useState('');
    const [confirmation, setConfirmation] = useState('');

    const run = async (name, call) => {
        setBusyAction(name);
        setError('');
        const result = await call();
        setBusyAction('');
        if (result && result.error) {
            setError(result.error);
            return false;
        }
        return true;
    };

    const confirm = async (name) => {
        setConfirmation('');
        const calls = {
            restart: () => actions.restartInstallation(installation.ID),
            unlock: () => actions.deletionUnlockInstallation(installation.ID),
            hibernate: () => actions.hibernateInstallation(installation.ID),
            delete: () => actions.deleteInstallation(installation.ID, installation.Name),
        };
        const ok = await run(name, calls[name]);
        if (ok && name === 'delete' && onDeleted) {
            onDeleted();
        }
    };

    return {
        busyAction,
        busy: busyAction !== '',
        error,
        confirmation,
        setConfirmation,
        run,
        confirm,
    };
}

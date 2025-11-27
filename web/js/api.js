// REST API calls for shell control

export async function resize(rows, cols) {
    try {
        await fetch('/resize', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ rows, cols })
        });
    } catch (error) {
        console.error('Failed to send terminal size:', error);
    }
}

export async function restart() {
    try {
        const response = await fetch('/restart', { method: 'POST' });
        return response.ok;
    } catch (error) {
        console.error('Restart error:', error);
        return false;
    }
}

export async function runCommand(cmd) {
    try {
        await fetch('/widget/lsh-sort/action', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                type: 'shell',
                cmd: cmd
            })
        });
    } catch (err) {
        console.error('Failed to run command:', err);
    }
}

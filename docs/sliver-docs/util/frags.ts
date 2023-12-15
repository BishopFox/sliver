/** URL Fragment Arguments */

class Frags {

    set(name: string, value: string) {
        let frags = this._parseHash();
        frags.set(name, value);
        let newFrags = "";
        for (let [key, value] of frags) {
            newFrags += `${key}=${value};`;
        }
        window.location.hash = newFrags;
    }

    get(name: string): string | undefined {
        if (typeof window === "undefined") {
            return undefined;
        }
        let frags = window.location.hash;
        if (!frags) {
            return undefined;
        }
        let fragsMap = this._parseHash();
        return fragsMap.get(name);
    }

    unset(name: string) {
        let frags = window.location.hash;
        if (!frags) {
            return;
        }
        let fragsMap = this._parseHash();
        fragsMap.delete(name);
        let newFrags = "";
        for (let [key, value] of fragsMap) {
            newFrags += `${key}=${value};`;
        }
        window.location.hash = newFrags;
    }

    private _parseHash(): Map<string, string> {
        let frags = window.location.hash;
        frags = frags.substring(1);
        let fragsMap = new Map<string, string>();
        if (!frags) {
            return fragsMap;
        }
        frags.split(";").forEach(frag => {
            if (!frag) {
                return;
            }
            const [key, value] = frag.split("=");
            fragsMap.set(key, value);
        });
        return fragsMap;
    }

}

export const frags = new Frags();

package service

import (
	"errors"
	"github.com/boltdb/bolt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/db"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	jsoniter "github.com/json-iterator/go"
	"strconv"
)

func GetPassagesByServer(tx *bolt.Tx, serverTicket string) (passages []model.Passage) {
	// server could be Server or Relay
	f := func(tx *bolt.Tx) error {
		bktTicket := tx.Bucket([]byte(model.BucketTicket))
		if bktTicket == nil {
			return nil
		}
		bktServer := tx.Bucket([]byte(model.BucketServer))
		if bktServer == nil {
			return nil
		}

		// get ticketObj of server
		bServerTicket := bktTicket.Get([]byte(serverTicket))
		if bServerTicket == nil {
			log.Warn("inconsistent: cannot find key %v in bucket %v", serverTicket, model.BucketTicket)
			return db.ErrKeyNotFound
		}
		var serverTicketObj model.Ticket
		if err := jsoniter.Unmarshal(bServerTicket, &serverTicketObj); err != nil {
			return err
		}
		// get serverObj of server
		bServer := bktServer.Get([]byte(serverTicket))
		if bServer == nil {
			log.Info("the server has not register yet: key %v", serverTicket)
			return db.ErrKeyNotFound
		}
		var serverObj model.Server
		if err := jsoniter.Unmarshal(bServer, &serverObj); err != nil {
			return err
		}
		if serverTicketObj.Type == model.TicketTypeServer {
			if serverObj.BandwidthLimit.Exhausted() {
				// do not generate Passages for exhausted servers
				return nil
			}
		}

		chatIdentifier := serverTicketObj.ChatIdentifier

		// generate all user/relay passages in this chat
		var userTickets []string
		var servers []model.Server
		var relays []model.Server
		_ = bktTicket.ForEach(func(k, v []byte) error {
			var ticket model.Ticket
			if err := jsoniter.Unmarshal(v, &ticket); err != nil {
				// do not stop the iter
				return nil
			}
			if ticket.ChatIdentifier != chatIdentifier {
				return nil
			}
			if common.Expired(ticket.ExpireAt) {
				return nil
			}
			// classify the ticket to slices above
			switch ticket.Type {
			case model.TicketTypeUser:
				// user ticket
				userTickets = append(userTickets, ticket.Ticket)
			case model.TicketTypeServer:
				// server ticket
				// only the relay need servers
				if serverTicketObj.Type == model.TicketTypeRelay {
					svr, err := GetServerByTicket(tx, ticket.Ticket)
					if err != nil {
						if !errors.Is(err, db.ErrKeyNotFound) {
							log.Warn("GetPassagesByServer: cannot get server by ticket: %v: %v", ticket.Ticket, err)
						}
						return nil
					}
					servers = append(servers, svr)
				}
			case model.TicketTypeRelay:
				// relay ticket
				// only the server need relays
				if serverTicketObj.Type == model.TicketTypeServer {
					relay, err := GetServerByTicket(tx, ticket.Ticket)
					if err != nil {
						if !errors.Is(err, db.ErrKeyNotFound) {
							log.Warn("GetPassagesByServer: cannot get server by ticket: %v: %v", ticket.Ticket, err)
						}
						return nil
					}
					relays = append(relays, relay)
				}
			}
			return nil
		})
		switch serverTicketObj.Type {
		case model.TicketTypeServer:
			// server inbounds are for users and relays
			for _, ticket := range userTickets {
				passages = append(passages, model.Passage{
					In: model.In{Argument: model.GetUserArgument(serverTicket, ticket, serverObj.Argument.Protocol)},
				})
			}
			if !serverObj.NoRelay {
				for _, relay := range relays {
					if relay.FailureCount >= model.MaxFailureCount || relay.BandwidthLimit.Exhausted() {
						continue
					}
					// TODO: splice different protocols
					if relay.Argument.Protocol != serverObj.Argument.Protocol {
						continue
					}
					passages = append(passages, model.Passage{
						In: model.In{
							From:     relay.Name,
							Argument: model.GetUserArgument(serverTicket, relay.Ticket, serverObj.Argument.Protocol),
						},
					})
				}
			}
		case model.TicketTypeRelay:
			// relay inbounds are for users but related with servers (n*m)
			// relay outbounds are for servers
			for _, svr := range servers {
				if svr.NoRelay {
					continue
				}
				if svr.FailureCount >= model.MaxFailureCount || svr.BandwidthLimit.Exhausted() {
					continue
				}
				// TODO: splice different protocols
				if svr.Argument.Protocol != serverObj.Argument.Protocol {
					continue
				}
				argRelayServer := model.GetUserArgument(svr.Ticket, serverTicket, serverObj.Argument.Protocol)
				for _, userTicket := range userTickets {
					argUserRelayServer := model.GetRelayUserArgument(svr.Ticket, serverTicket, userTicket, serverObj.Argument.Protocol)
					passages = append(passages, model.Passage{
						In: model.In{Argument: argUserRelayServer},
						Out: &model.Out{
							To: svr.Name,
							// FIXME: Relay only connects to the first host of the endpoint server
							Host:     model.GetFirstHost(svr.Hosts),
							Port:     strconv.Itoa(svr.Port),
							Argument: argRelayServer,
						},
					})
				}
			}
		}
		return nil
	}
	if tx != nil {
		f(tx)
		return passages
	}
	db.DB().View(f)
	return passages
}
